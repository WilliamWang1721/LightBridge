//go:build unit

package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserHandlerListAddsActivityFilterToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	handler := NewUserHandler(adminSvc, nil, nil, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/users?search=example.com&activity=none",
		nil,
	)

	handler.List(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "example.com @activity:none", adminSvc.lastListUsers.filters.Search)
}

func TestUserHandlerListRejectsInvalidActivityFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	handler := NewUserHandler(adminSvc, nil, nil, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/users?activity=recent",
		nil,
	)

	handler.List(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Empty(t, adminSvc.lastListUsers.filters.Search)
}
