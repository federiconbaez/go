package grpc

import (
	"context"
	"fmt"
	"io"
	"time"

	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/application/usecases"
	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/entities"
	https://github.com/federiconbaez/gogrpc-go-android/server-go/internal/domain/ports"
	pb https://github.com/federiconbaez/gogrpc-go-android/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NotebookServer implementa el servidor gRPC para el servicio de cuaderno
type NotebookServer struct {
	pb.UnimplementedNotebookServiceServer
	ideaUseCases     *usecases.IdeaUseCases
	reminderUseCases *usecases.ReminderUseCases
	fileUseCases     *usecases.FileUseCases
	progressUseCases *usecases.ProgressUseCases
	notificationSvc  ports.NotificationService
}

// NewNotebookServer crea una nueva instancia del servidor gRPC
func NewNotebookServer(
	ideaUseCases *usecases.IdeaUseCases,
	reminderUseCases *usecases.ReminderUseCases,
	fileUseCases *usecases.FileUseCases,
	progressUseCases *usecases.ProgressUseCases,
	notificationSvc ports.NotificationService,
) *NotebookServer {
	return &NotebookServer{
		ideaUseCases:     ideaUseCases,
		reminderUseCases: reminderUseCases,
		fileUseCases:     fileUseCases,
		progressUseCases: progressUseCases,
		notificationSvc:  notificationSvc,
	}
}

// CreateIdea implementa la creación de ideas
func (s *NotebookServer) CreateIdea(ctx context.Context, req *pb.CreateIdeaRequest) (*pb.CreateIdeaResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return &pb.CreateIdeaResponse{
			Success: false,
			Message: "Invalid user ID format",
		}, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	idea, err := s.ideaUseCases.CreateIdea(
		ctx,
		req.Title,
		req.Content,
		entities.IdeaCategory(req.Category),
		userID,
		req.Tags,
		req.Priority,
	)
	if err != nil {
		return &pb.CreateIdeaResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create idea: %v", err),
		}, status.Error(codes.Internal, err.Error())
	}

	return &pb.CreateIdeaResponse{
		Idea:    s.convertIdeaToProto(idea),
		Success: true,
		Message: "Idea created successfully",
	}, nil
}

// GetIdea implementa la obtención de ideas
func (s *NotebookServer) GetIdea(ctx context.Context, req *pb.GetIdeaRequest) (*pb.GetIdeaResponse, error) {
	ideaID, err := uuid.Parse(req.Id)
	if err != nil {
		return &pb.GetIdeaResponse{
			Success: false,
			Message: "Invalid idea ID format",
		}, status.Error(codes.InvalidArgument, "invalid idea ID")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return &pb.GetIdeaResponse{
			Success: false,
			Message: "Invalid user ID format",
		}, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	idea, err := s.ideaUseCases.GetIdea(ctx, ideaID, userID)
	if err != nil {
		if err == entities.ErrIdeaNotFound {
			return &pb.GetIdeaResponse{
				Success: false,
				Message: "Idea not found",
			}, status.Error(codes.NotFound, "idea not found")
		}
		if err == entities.ErrIdeaUnauthorized {
			return &pb.GetIdeaResponse{
				Success: false,
				Message: "Unauthorized access to idea",
			}, status.Error(codes.PermissionDenied, "unauthorized")
		}
		return &pb.GetIdeaResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get idea: %v", err),
		}, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetIdeaResponse{
		Idea:    s.convertIdeaToProto(idea),
		Success: true,
		Message: "Idea retrieved successfully",
	}, nil
}

// ListIdeas implementa la lista de ideas
func (s *NotebookServer) ListIdeas(ctx context.Context, req *pb.ListIdeasRequest) (*pb.ListIdeasResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return &pb.ListIdeasResponse{
			Success: false,
			Message: "Invalid user ID format",
		}, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	filters := ports.IdeaFilters{
		Category: entities.IdeaCategory(req.Category),
		Status:   entities.IdeaStatus(req.Status),
		Tags:     req.Tags,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
	}

	// Valores por defecto para paginación
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PageSize <= 0 {
		filters.PageSize = 10
	}

	ideas, totalCount, err := s.ideaUseCases.ListIdeas(ctx, userID, filters)
	if err != nil {
		return &pb.ListIdeasResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to list ideas: %v", err),
		}, status.Error(codes.Internal, err.Error())
	}

	protoIdeas := make([]*pb.Idea, len(ideas))
	for i, idea := range ideas {
		protoIdeas[i] = s.convertIdeaToProto(idea)
	}

	return &pb.ListIdeasResponse{
		Ideas:      protoIdeas,
		TotalCount: int32(totalCount),
		Page:       int32(filters.Page),
		PageSize:   int32(filters.PageSize),
		Success:    true,
		Message:    "Ideas retrieved successfully",
	}, nil
}

// UpdateIdea implementa la actualización de ideas
func (s *NotebookServer) UpdateIdea(ctx context.Context, req *pb.UpdateIdeaRequest) (*pb.UpdateIdeaResponse, error) {
	ideaID, err := uuid.Parse(req.Id)
	if err != nil {
		return &pb.UpdateIdeaResponse{
			Success: false,
			Message: "Invalid idea ID format",
		}, status.Error(codes.InvalidArgument, "invalid idea ID")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return &pb.UpdateIdeaResponse{
			Success: false,
			Message: "Invalid user ID format",
		}, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	idea, err := s.ideaUseCases.UpdateIdea(
		ctx,
		ideaID,
		userID,
		req.Title,
		req.Content,
		req.Tags,
		entities.IdeaCategory(req.Category),
		entities.IdeaStatus(req.Status),
		req.Priority,
	)
	if err != nil {
		if err == entities.ErrIdeaNotFound {
			return &pb.UpdateIdeaResponse{
				Success: false,
				Message: "Idea not found",
			}, status.Error(codes.NotFound, "idea not found")
		}
		if err == entities.ErrIdeaUnauthorized {
			return &pb.UpdateIdeaResponse{
				Success: false,
				Message: "Unauthorized access to idea",
			}, status.Error(codes.PermissionDenied, "unauthorized")
		}
		return &pb.UpdateIdeaResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to update idea: %v", err),
		}, status.Error(codes.Internal, err.Error())
	}

	return &pb.UpdateIdeaResponse{
		Idea:    s.convertIdeaToProto(idea),
		Success: true,
		Message: "Idea updated successfully",
	}, nil
}

// DeleteIdea implementa la eliminación de ideas
func (s *NotebookServer) DeleteIdea(ctx context.Context, req *pb.DeleteIdeaRequest) (*pb.DeleteIdeaResponse, error) {
	ideaID, err := uuid.Parse(req.Id)
	if err != nil {
		return &pb.DeleteIdeaResponse{
			Success: false,
			Message: "Invalid idea ID format",
		}, status.Error(codes.InvalidArgument, "invalid idea ID")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return &pb.DeleteIdeaResponse{
			Success: false,
			Message: "Invalid user ID format",
		}, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	err = s.ideaUseCases.DeleteIdea(ctx, ideaID, userID)
	if err != nil {
		if err == entities.ErrIdeaNotFound {
			return &pb.DeleteIdeaResponse{
				Success: false,
				Message: "Idea not found",
			}, status.Error(codes.NotFound, "idea not found")
		}
		if err == entities.ErrIdeaUnauthorized {
			return &pb.DeleteIdeaResponse{
				Success: false,
				Message: "Unauthorized access to idea",
			}, status.Error(codes.PermissionDenied, "unauthorized")
		}
		return &pb.DeleteIdeaResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete idea: %v", err),
		}, status.Error(codes.Internal, err.Error())
	}

	return &pb.DeleteIdeaResponse{
		Success: true,
		Message: "Idea deleted successfully",
	}, nil
}

// UploadFile implementa la subida de archivos con streaming
func (s *NotebookServer) UploadFile(stream pb.NotebookService_UploadFileServer) error {
	var metadata *pb.FileMetadata
	var fileData []byte

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Error(codes.Internal, fmt.Sprintf("Failed to receive chunk: %v", err))
		}

		switch data := req.Data.(type) {
		case *pb.UploadFileRequest_Metadata:
			metadata = data.Metadata
		case *pb.UploadFileRequest_Chunk:
			fileData = append(fileData, data.Chunk...)
		}
	}

	if metadata == nil {
		return status.Error(codes.InvalidArgument, "File metadata is required")
	}

	userID, err := uuid.Parse(metadata.UserId)
	if err != nil {
		return status.Error(codes.InvalidArgument, "Invalid user ID format")
	}

	// Crear un reader desde los datos del archivo
	reader := &bytesReader{data: fileData}

	fileInfo, err := s.fileUseCases.UploadFile(
		stream.Context(),
		metadata.Filename,
		metadata.ContentType,
		reader,
		userID,
		metadata.Compress,
		metadata.CompressionType,
	)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Failed to upload file: %v", err))
	}

	response := &pb.UploadFileResponse{
		FileInfo: s.convertFileInfoToProto(fileInfo),
		Success:  true,
		Message:  "File uploaded successfully",
		UploadId: fileInfo.ID.String(),
	}

	return stream.SendAndClose(response)
}

// SubscribeNotifications implementa la suscripción a notificaciones
func (s *NotebookServer) SubscribeNotifications(req *pb.NotificationSubscriptionRequest, stream pb.NotebookService_SubscribeNotificationsServer) error {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return status.Error(codes.InvalidArgument, "Invalid user ID format")
	}

	notificationCh, err := s.notificationSvc.SubscribeToNotifications(stream.Context(), userID, req.Channels)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("Failed to subscribe to notifications: %v", err))
	}

	for {
		select {
		case notification := <-notificationCh:
			protoNotification := &pb.NotificationResponse{
				Id:        notification.ID.String(),
				Title:     notification.Title,
				Message:   notification.Message,
				Type:      notification.Type,
				CreatedAt: timestamppb.New(time.Now()),
				UserId:    userID.String(),
				Metadata:  notification.Metadata,
			}
			if err := stream.Send(protoNotification); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// Métodos auxiliares para conversiones

func (s *NotebookServer) convertIdeaToProto(idea *entities.Idea) *pb.Idea {
	relatedIdeas := make([]string, len(idea.RelatedIdeas))
	for i, id := range idea.RelatedIdeas {
		relatedIdeas[i] = id.String()
	}

	return &pb.Idea{
		Id:           idea.ID.String(),
		Title:        idea.Title,
		Content:      idea.Content,
		Tags:         idea.Tags,
		Category:     pb.IdeaCategory(idea.Category),
		Status:       pb.IdeaStatus(idea.Status),
		CreatedAt:    timestamppb.New(idea.CreatedAt),
		UpdatedAt:    timestamppb.New(idea.UpdatedAt),
		UserId:       idea.UserID.String(),
		RelatedIdeas: relatedIdeas,
		Priority:     idea.Priority,
	}
}

func (s *NotebookServer) convertFileInfoToProto(fileInfo *entities.FileInfo) *pb.FileInfo {
	return &pb.FileInfo{
		Id:              fileInfo.ID.String(),
		Filename:        fileInfo.Filename,
		ContentType:     fileInfo.ContentType,
		Size:            fileInfo.Size,
		Checksum:        fileInfo.Checksum,
		CreatedAt:       timestamppb.New(fileInfo.CreatedAt),
		UserId:          fileInfo.UserID.String(),
		Compressed:      fileInfo.Compressed,
		CompressionType: fileInfo.CompressionType,
		Path:            fileInfo.Path,
	}
}

// bytesReader es un helper para convertir []byte a io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}