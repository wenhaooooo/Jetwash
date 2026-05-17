package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"jetwash/pkg/ecode"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestOK(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		OK(c, gin.H{"key": "value"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, ecode.Success, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)
}

func TestError(t *testing.T) {
	tests := []struct {
		name       string
		ecode      ecode.Ecode
		wantStatus int
	}{
		{"unauthorized", ecode.ErrUnauthorized, http.StatusUnauthorized},
		{"forbidden", ecode.ErrForbidden, http.StatusForbidden},
		{"not found", ecode.ErrNotFound, http.StatusNotFound},
		{"invalid params", ecode.ErrInvalidParams, http.StatusBadRequest},
		{"server error", ecode.ErrServer, http.StatusInternalServerError},
		{"too many requests", ecode.ErrTooManyRequests, http.StatusTooManyRequests},
		{"tenant not found", ecode.ErrTenantNotFound, http.StatusNotFound},
		{"invalid api key", ecode.ErrInvalidAPIKey, http.StatusUnauthorized},
		{"request timeout", ecode.ErrRequestTimeout, http.StatusRequestTimeout},
		{"tenant inactive", ecode.ErrTenantInactive, http.StatusForbidden},
		{"tenant suspended", ecode.ErrTenantSuspended, http.StatusForbidden},
		{"word not found", ecode.ErrWordNotFound, http.StatusNotFound},
		{"word already exists", ecode.ErrWordAlreadyExists, http.StatusBadRequest},
		{"text too long", ecode.ErrTextTooLong, http.StatusBadRequest},
		{"conflict", ecode.ErrConflict, http.StatusConflict},
		{"default to 500", ecode.New(9999, "unknown error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/test", func(c *gin.Context) {
				Error(c, tt.ecode)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.ecode.Code(), resp.Code)
			assert.Equal(t, tt.ecode.Message(), resp.Message)
		})
	}
}

func TestOKWithMessage(t *testing.T) {
	router := setupTestRouter()
	router.GET("/test", func(c *gin.Context) {
		OKWithMessage(c, "custom success", gin.H{"key": "value"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, ecode.Success, resp.Code)
	assert.Equal(t, "custom success", resp.Message)
	assert.NotNil(t, resp.Data)
}

func TestErrorWithMessage(t *testing.T) {
	tests := []struct {
		name       string
		ecode      ecode.Ecode
		message    string
		wantStatus int
	}{
		{"overrides forbidden message", ecode.ErrForbidden, "access denied to resource", http.StatusForbidden},
		{"overrides not found message", ecode.ErrNotFound, "item does not exist", http.StatusNotFound},
		{"overrides server error message", ecode.ErrServer, "something went wrong", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.GET("/test", func(c *gin.Context) {
				ErrorWithMessage(c, tt.ecode, tt.message)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp Response
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, tt.ecode.Code(), resp.Code)
			assert.Equal(t, tt.message, resp.Message)
		})
	}
}

func TestCreated(t *testing.T) {
	router := setupTestRouter()
	router.POST("/test", func(c *gin.Context) {
		Created(c, gin.H{"id": 1})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, ecode.Success, resp.Code)
}
