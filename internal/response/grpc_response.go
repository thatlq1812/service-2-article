package response

import (
	pb "github.com/thatlq1812/service-2-article/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Standard response codes mapping
const (
	CodeSuccess            = "000" // Success
	CodeUnknownError       = "002" // Unknown error
	CodeInvalidRequest     = "003" // Invalid request
	CodeNotFound           = "005" // Not found
	CodeAlreadyExists      = "006" // Already exists
	CodePermissionDenied   = "007" // Permission denied
	CodeInternalError      = "013" // Internal error
	CodeUnauthenticated    = "014" // Authentication required
	CodeServiceUnavailable = "015" // Service unavailable
	CodeUnauthorized       = "016" // Unauthorized
)

// Article Service Response Helpers

func CreateArticleSuccess(article *pb.Article) *pb.CreateArticleResponse {
	return &pb.CreateArticleResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data: &pb.CreateArticleData{
			Article: article,
		},
	}
}

func GetArticleSuccess(article *pb.ArticleWithUser) *pb.GetArticleResponse {
	return &pb.GetArticleResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data: &pb.GetArticleData{
			Article: article,
		},
	}
}

// GetArticleSuccessWithMessage returns a success response with custom message
// Used for graceful degradation scenarios (e.g., author info unavailable)
func GetArticleSuccessWithMessage(article *pb.ArticleWithUser, message string) *pb.GetArticleResponse {
	return &pb.GetArticleResponse{
		Code:    CodeSuccess,
		Message: message,
		Data: &pb.GetArticleData{
			Article: article,
		},
	}
}

func UpdateArticleSuccess(article *pb.Article) *pb.UpdateArticleResponse {
	return &pb.UpdateArticleResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data: &pb.UpdateArticleData{
			Article: article,
		},
	}
}

func DeleteArticleSuccess() *pb.DeleteArticleResponse {
	return &pb.DeleteArticleResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data: &pb.DeleteArticleData{
			Success: true,
		},
	}
}

func ListArticlesSuccess(articles []*pb.ArticleWithUser, total, page, totalPages int32) *pb.ListArticlesResponse {
	return &pb.ListArticlesResponse{
		Code:    CodeSuccess,
		Message: "success",
		Data: &pb.ListArticlesData{
			Articles:   articles,
			Total:      total,
			Page:       page,
			TotalPages: totalPages,
		},
	}
}

// Error response helpers - return wrapped responses with error codes

// CreateArticleError returns error response for CreateArticle
func CreateArticleError(code codes.Code, message string) *pb.CreateArticleResponse {
	return &pb.CreateArticleResponse{
		Code:    MapGRPCCodeToString(code),
		Message: message,
		Data:    nil,
	}
}

// GetArticleError returns error response for GetArticle
func GetArticleError(code codes.Code, message string) *pb.GetArticleResponse {
	return &pb.GetArticleResponse{
		Code:    MapGRPCCodeToString(code),
		Message: message,
		Data:    nil,
	}
}

// UpdateArticleError returns error response for UpdateArticle
func UpdateArticleError(code codes.Code, message string) *pb.UpdateArticleResponse {
	return &pb.UpdateArticleResponse{
		Code:    MapGRPCCodeToString(code),
		Message: message,
		Data:    nil,
	}
}

// DeleteArticleError returns error response for DeleteArticle
func DeleteArticleError(code codes.Code, message string) *pb.DeleteArticleResponse {
	return &pb.DeleteArticleResponse{
		Code:    MapGRPCCodeToString(code),
		Message: message,
		Data:    nil,
	}
}

// ListArticlesError returns error response for ListArticles
func ListArticlesError(code codes.Code, message string) *pb.ListArticlesResponse {
	return &pb.ListArticlesResponse{
		Code:    MapGRPCCodeToString(code),
		Message: message,
		Data:    nil,
	}
}

// MapGRPCCodeToString converts gRPC code to string code
func MapGRPCCodeToString(code codes.Code) string {
	switch code {
	case codes.OK:
		return CodeSuccess
	case codes.InvalidArgument:
		return CodeInvalidRequest
	case codes.NotFound:
		return CodeNotFound
	case codes.AlreadyExists:
		return CodeAlreadyExists
	case codes.PermissionDenied:
		return CodePermissionDenied
	case codes.Unauthenticated:
		return CodeUnauthenticated
	case codes.Unavailable:
		return CodeServiceUnavailable
	case codes.Internal:
		return CodeInternalError
	default:
		return CodeUnknownError
	}
}

// GRPCError creates a standardized gRPC error with hints
func GRPCError(code codes.Code, message string) error {
	// Add hints based on code
	hint := ""
	switch code {
	case codes.InvalidArgument:
		hint = " Check input parameters for validity."
	case codes.NotFound:
		hint = " Verify the resource ID exists."
	case codes.Unauthenticated:
		hint = " Provide valid authentication credentials."
	case codes.PermissionDenied:
		hint = " Ensure you have the required permissions."
	case codes.Internal:
		hint = " Contact support if the issue persists."
	default:
		hint = ""
	}
	fullMessage := message + hint
	return status.Error(code, fullMessage)
}

// GRPCErrorWithCode is an alias for GRPCError for backward compatibility
func GRPCErrorWithCode(code codes.Code, message string) error {
	return GRPCError(code, message)
}
