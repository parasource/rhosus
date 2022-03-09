package registry

import (
	"context"
	api_pb "github.com/parasource/rhosus/rhosus/pb/api"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"sort"
	"strings"
)

type APIConfig struct {
	Host string
	Port string
}

type API struct {
	api_pb.ApiServer

	config APIConfig
	r      *Registry
}

func NewAPIServer(r *Registry, config APIConfig) (*API, error) {
	a := &API{
		r:      r,
		config: config,
	}

	address := net.JoinHostPort(a.config.Host, a.config.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	api_pb.RegisterApiServer(grpcServer, a)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {

		}
	}()

	go func() {
		select {
		case <-a.r.NotifyShutdown():
			err := lis.Close()
			if err != nil {
				logrus.Errorf("error closing api server tcp: %v", err)
			}

			return
		}
	}()

	return a, nil
}

func (a *API) Shutdown() {
	// todo
}

func (a *API) Ping(ctx context.Context, r *api_pb.Void) (*api_pb.Void, error) {
	return &api_pb.Void{}, nil
}

func (a *API) MakeDir(ctx context.Context, r *api_pb.MakeDirRequest) (*api_pb.CommonResponse, error) {

	path := strings.Trim(r.Path, "/")

	dir, err := a.r.MemoryStorage.GetFileByPath(path)
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
		Size_:    0,
	}
	sPath := strings.Split(path, "/")
	if len(sPath) > 1 {
		parent, err := a.r.MemoryStorage.GetFileByPath(strings.Join(sPath[:len(sPath)-1], "/"))
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

	err = a.r.MemoryStorage.StoreFile(file)
	if err != nil {
		return nil, err
	}

	return &api_pb.CommonResponse{Success: true}, nil
}

func (a *API) Remove(ctx context.Context, r *api_pb.RemoveRequest) (*api_pb.CommonResponse, error) {
	path := strings.Trim(r.Path, "/")

	rootFile, err := a.r.MemoryStorage.GetFileByPath(path)
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

	err = a.killChildren(rootFile)
	if err != nil {
		return nil, err
	}
	if rootFile.Type == control_pb.FileInfo_FILE {
		logrus.Infof("removing file blocks: %v", rootFile)
		err, errs := a.r.RemoveFileBlocks(rootFile)
		if err != nil {
			return nil, err
		}
		if errs != nil && len(errs) > 0 {
			// todo
		}
	}

	return &api_pb.CommonResponse{
		Success: true,
	}, nil
}

func (a *API) killChildren(file *control_pb.FileInfo) error {
	childFiles, err := a.r.MemoryStorage.GetFilesByParentId(file.Id)
	if err != nil {
		return err
	}
	if len(childFiles) == 0 {
		return nil
	}

	if file.Type == control_pb.FileInfo_FILE {
		err, errs := a.r.RemoveFileBlocks(file)
		if err != nil {
			return err
		}
		if errs != nil && len(errs) > 0 {
			for nodeID, nodeErr := range errs {
				logrus.Errorf("error deleting blocks from node %v: %v", nodeID, nodeErr)
			}
		}
	}

	for _, childFile := range childFiles {
		err := a.killChildren(childFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *API) List(ctx context.Context, r *api_pb.ListRequest) (*api_pb.ListResponse, error) {

	var parentID string
	if r.Path == "/" {
		parentID = "root"
	} else {
		dir, err := a.r.MemoryStorage.GetFileByPath(strings.Trim(r.Path, "/"))
		if err != nil {
			return nil, err
		}
		if dir == nil {
			return &api_pb.ListResponse{Error: "no such file or directory"}, nil
		}
		parentID = dir.Id
	}

	var list []*api_pb.FileInfo

	files, err := a.r.MemoryStorage.GetFilesByParentId(parentID)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		info := &api_pb.FileInfo{
			Name:  file.Name,
			Size_: file.Size_,
		}
		switch file.Type {
		case control_pb.FileInfo_DIR:
			info.Type = api_pb.FileInfo_DIR
		case control_pb.FileInfo_FILE:
			info.Type = api_pb.FileInfo_FILE
		}
		list = append(list, info)

	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name[0] < list[j].Name[1]
	})
	return &api_pb.ListResponse{
		List: list,
	}, nil
}

func (a *API) Upload(srv api_pb.Api_UploadServer) error {

	return nil
}

func (a *API) Download(r *api_pb.DownloadRequest, stream api_pb.Api_DownloadServer) error {

	return nil
}
