/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package registry

import (
	"fmt"
	api_pb "github.com/parasource/rhosus/rhosus/pb/api"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/rs/zerolog/log"
	"strings"
)

func (r *Registry) HandleMakeDir(req *api_pb.MakeDirRequest) (*api_pb.CommonResponse, error) {
	path := strings.Trim(req.Path, "/")

	dir, err := r.Storage.GetFileByPath(path)
	if err != nil {
		return nil, err
	}
	// if dir already exists
	if dir != nil {
		return &api_pb.CommonResponse{
			Success: false,
			Err:     "directory already exists",
		}, nil
	}

	uid, _ := uuid.NewV4()
	filePathSplit := strings.Split(path, "/")
	fileName := filePathSplit[len(filePathSplit)-1]
	file := &control_pb.FileInfo{
		Id:       uid.String(),
		Name:     fileName,
		ParentID: "root",
		Type:     control_pb.FileInfo_DIR,
		Path:     path,
		FileSize: 0,
	}
	sPath := strings.Split(path, "/")
	if len(sPath) > 1 {
		parent, err := r.Storage.GetFileByPath(strings.Join(sPath[:len(sPath)-1], "/"))
		if parent == nil {
			return &api_pb.CommonResponse{
				Success: false,
				Err:     "no such file or directory",
			}, nil
		}
		if err != nil {
			return nil, err
		}
		file.ParentID = parent.Id
	}

	err = r.Storage.StoreFile(file)
	if err != nil {
		return nil, err
	}

	err = r.Cluster.WriteAssignFileEntry(file)
	if err != nil {
		// todo delete file
		return nil, fmt.Errorf("error writing file assign entry: %w", err)
	}

	return &api_pb.CommonResponse{Success: true}, nil
}

func (r *Registry) HandleRemoveFileOrPath(req *api_pb.RemoveRequest) (*api_pb.CommonResponse, error) {
	path := strings.Trim(req.Path, "/")

	rootFile, err := r.Storage.GetFileByPath(path)
	if err != nil {
		return nil, err
	}
	// if dir already exists
	if rootFile == nil {
		return &api_pb.CommonResponse{
			Success: false,
			Err:     "no such file or directory",
		}, nil
	}

	blocks, err := r.Storage.GetBlocks(rootFile.Id)
	if err != nil {
		return nil, err
	}

	err = r.killChildren(rootFile)
	if err != nil {
		return nil, err
	}
	if rootFile.Type == control_pb.FileInfo_FILE {
		err, errs := r.RemoveFileBlocks(rootFile)
		if err != nil {
			return nil, err
		}
		if errs != nil && len(errs) > 0 {
			// todo
		}
	}

	err = r.Cluster.WriteDeleteFileEntry(rootFile)
	if err != nil {
		// todo revert changes
		return nil, err
	}
	err = r.Cluster.WriteDeleteBlocksEntry(blocks)
	if err != nil {
		// todo revert changes
		return nil, err
	}

	return &api_pb.CommonResponse{
		Success: true,
	}, nil
}

func (r *Registry) killChildren(file *control_pb.FileInfo) error {
	childFiles, err := r.Storage.GetFilesByParentId(file.Id)
	if err != nil {
		return err
	}
	if len(childFiles) == 0 {
		return nil
	}

	if file.Type == control_pb.FileInfo_FILE {
		err, errs := r.RemoveFileBlocks(file)
		if err != nil {
			return err
		}
		if errs != nil && len(errs) > 0 {
			for nodeID, nodeErr := range errs {
				log.Error().Err(nodeErr).Str("node_id", nodeID).Msg("error deleting blocks from node")
			}
		}
	}

	for _, childFile := range childFiles {
		err := r.killChildren(childFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) HandleCreatePolicy(req *api_pb.CreatePolicyRequest) (*api_pb.CreatePolicyResponse, error) {
	paths := make([]*control_pb.Policy_PathRules, 0, len(req.Paths))
	for _, path := range req.Paths {
		paths = append(paths, &control_pb.Policy_PathRules{
			Path:               path.Path,
			Policy:             req.Name,
			CapabilitiesBitmap: 0,
			Capabilities:       path.Capabilities,
		})
	}
	uid, _ := uuid.NewV4()
	policy := &control_pb.Policy{
		Id:    uid.String(),
		Name:  req.Name,
		Paths: paths,
	}

	err := r.Storage.StorePolicy(policy)
	if err != nil {
		return nil, err
	}

	return &api_pb.CreatePolicyResponse{}, nil
}

func (r *Registry) HandleListPolicies(req *api_pb.ListPoliciesRequest) (*api_pb.ListPoliciesResponse, error) {
	policies, err := r.Storage.ListPolicies()
	if err != nil {
		return nil, fmt.Errorf("error listing policies: %w", err)
	}

	var res api_pb.ListPoliciesResponse
	for _, policy := range policies {
		paths := make([]*api_pb.PolicyPathRules, 0, len(policy.Paths))
		for _, path := range policy.Paths {
			paths = append(paths, &api_pb.PolicyPathRules{
				Path:         path.Path,
				Capabilities: path.Capabilities,
			})
		}
		res.Policies = append(res.Policies, &api_pb.Policy{
			Name:  policy.Name,
			Paths: paths,
		})
	}

	return &res, nil
}
