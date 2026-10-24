package api

import "net/http"

// route is one HTTP route registration.
//
// Routes are declared in a single table so that Handler and the contract
// consistency test (contract_test.go) share one source of truth.
type route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}

// routes returns every API route, in registration order.
func (s *Server) routes() []route {
	return []route{
		// System
		{http.MethodGet, "/healthz", s.handleHealth},
		{http.MethodGet, "/api/v1/version", s.handleVersion},

		// Auth
		{http.MethodGet, "/api/v1/auth/status", s.optionalAuth(s.handleAuthStatus)},
		{http.MethodPost, "/api/v1/setup", s.handleSetup},
		{http.MethodPost, "/api/v1/auth/login", s.handleLogin},
		{http.MethodPost, "/api/v1/auth/logout", s.requireAuth(s.handleLogout)},
		{http.MethodGet, "/api/v1/auth/me", s.requireAuth(s.handleMe)},
		{http.MethodPut, "/api/v1/auth/password", s.requireAuth(s.handleChangePassword)},

		// Public
		{http.MethodGet, "/api/v1/public/navigation", s.publicRead(s.handlePublicNavigation)},
		{http.MethodGet, "/api/v1/public/categories", s.publicRead(s.handlePublicCategories)},
		{http.MethodGet, "/api/v1/public/categories/{id}/bookmarks", s.publicRead(s.handlePublicCategoryBookmarks)},
		{http.MethodGet, "/api/v1/public/search", s.publicRead(s.handlePublicSearch)},
		{http.MethodGet, "/api/v1/public/settings", s.publicRead(s.handlePublicSettings)},
		{http.MethodGet, "/api/v1/public/uploads/{filename}", s.handlePublicUpload},

		// Admin: categories
		{http.MethodGet, "/api/v1/admin/categories", s.requireAuth(s.handleListCategories)},
		{http.MethodPost, "/api/v1/admin/categories", s.requireAuth(s.handleCreateCategory)},
		{http.MethodPut, "/api/v1/admin/categories/order", s.requireAuth(s.handleReorderCategories)},
		{http.MethodGet, "/api/v1/admin/categories/{id}", s.requireAuth(s.handleGetCategory)},
		{http.MethodPatch, "/api/v1/admin/categories/{id}", s.requireAuth(s.handleUpdateCategory)},
		{http.MethodDelete, "/api/v1/admin/categories/{id}", s.requireAuth(s.handleDeleteCategory)},

		// Admin: bookmarks
		{http.MethodGet, "/api/v1/admin/bookmarks", s.requireAuth(s.handleListBookmarks)},
		{http.MethodPost, "/api/v1/admin/bookmarks", s.requireAuth(s.handleCreateBookmark)},
		{http.MethodPut, "/api/v1/admin/bookmarks/order", s.requireAuth(s.handleReorderBookmarks)},
		{http.MethodGet, "/api/v1/admin/bookmarks/{id}", s.requireAuth(s.handleGetBookmark)},
		{http.MethodPatch, "/api/v1/admin/bookmarks/{id}", s.requireAuth(s.handleUpdateBookmark)},
		{http.MethodDelete, "/api/v1/admin/bookmarks/{id}", s.requireAuth(s.handleDeleteBookmark)},
		{http.MethodPost, "/api/v1/admin/bookmarks/{id}/move", s.requireAuth(s.handleMoveBookmark)},

		// Admin: settings, icons, transfer, backups
		{http.MethodGet, "/api/v1/admin/settings", s.requireAuth(s.handleGetSettings)},
		{http.MethodPut, "/api/v1/admin/settings", s.requireAuth(s.handleUpdateSettings)},
		{http.MethodPost, "/api/v1/admin/uploads/icon", s.requireAuth(s.handleUploadIcon)},
		{http.MethodPost, "/api/v1/admin/icons/fetch", s.requireAuth(s.handleFetchIcon)},
		{http.MethodGet, "/api/v1/admin/export", s.requireAuth(s.handleExport)},
		{http.MethodPost, "/api/v1/admin/import/preview", s.requireAuth(s.handleImportPreview)},
		{http.MethodPost, "/api/v1/admin/import", s.requireAuth(s.handleImport)},
		{http.MethodGet, "/api/v1/admin/backups", s.requireAuth(s.handleListBackups)},
		{http.MethodPost, "/api/v1/admin/backups", s.requireAuth(s.handleCreateBackup)},
		{http.MethodPost, "/api/v1/admin/backups/{id}/restore", s.requireAuth(s.handleRestoreBackup)},
		{http.MethodGet, "/api/v1/admin/update-check", s.requireAuth(s.handleUpdateCheck)},
	}
}
