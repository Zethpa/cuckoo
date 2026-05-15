package handlers

import (
	"net/http"
	"strconv"

	"cuckoo/backend/internal/auth"
	"cuckoo/backend/internal/config"
	"cuckoo/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	cfg   config.Config
	auth  *services.AuthService
	rooms *services.RoomService
}

func New(cfg config.Config, authSvc *services.AuthService, roomSvc *services.RoomService) *Handler {
	return &Handler{cfg: cfg, auth: authSvc, rooms: roomSvc}
}

func (h *Handler) Register(r *gin.Engine, ws gin.HandlerFunc) {
	r.Use(cors(h.cfg))
	api := r.Group("/api")
	api.POST("/auth/login", h.login)
	api.POST("/auth/logout", h.logout)
	protected := api.Group("")
	protected.Use(h.AuthMiddleware())
	protected.GET("/me", h.me)
	protected.POST("/me/password", h.changePassword)
	protected.GET("/me/games", h.myGames)
	protected.GET("/admin/users", h.AdminMiddleware(), h.listUsers)
	protected.POST("/admin/users", h.AdminMiddleware(), h.createUser)
	protected.DELETE("/admin/users/:id", h.AdminMiddleware(), h.disableUser)
	protected.POST("/admin/users/:id/restore", h.AdminMiddleware(), h.restoreUser)
	protected.POST("/admin/users/:id/reset-password", h.AdminMiddleware(), h.resetPassword)
	protected.POST("/rooms", h.createRoom)
	protected.POST("/rooms/join", h.joinRoom)
	protected.GET("/rooms/:code", h.room)
	protected.GET("/games/:code", h.gameArchive)
	protected.POST("/rooms/:code/ready", h.ready)
	protected.PATCH("/rooms/:code/settings", h.settings)
	protected.POST("/rooms/:code/start-roll", h.startRoll)
	protected.POST("/rooms/:code/roll", h.roll)
	protected.POST("/rooms/:code/start-game", h.startGame)
	protected.POST("/rooms/:code/contributions", h.contribution)
	protected.GET("/ws/rooms/:code", ws)
}

func (h *Handler) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet("claims").(*auth.Claims)
		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin only"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(auth.CookieName)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		claims, err := auth.ParseToken(h.cfg, token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}

func (h *Handler) login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	user, token, err := h.auth.Login(req.Username, req.Password)
	if err != nil {
		fail(c, http.StatusUnauthorized, err)
		return
	}
	auth.SetCookie(c, h.cfg, token)
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) logout(c *gin.Context) {
	auth.ClearCookie(c, h.cfg)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) me(c *gin.Context) {
	claims := c.MustGet("claims").(*auth.Claims)
	user, err := h.auth.FindUser(claims.UserID)
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) myGames(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	games, err := h.rooms.ListUserGames(userID(c), limit)
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"games": games})
}

func (h *Handler) listUsers(c *gin.Context) {
	users, err := h.auth.ListUsers()
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *Handler) createUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	initialPassword, err := h.auth.AddUserWithGeneratedPassword(req.Username, req.Role)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	users, err := h.auth.ListUsers()
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"users": users, "initialPassword": initialPassword})
}

func (h *Handler) disableUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.auth.DisableUser(userID(c), uint(id)); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	users, err := h.auth.ListUsers()
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *Handler) restoreUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.auth.RestoreUser(uint(id)); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	users, err := h.auth.ListUsers()
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *Handler) resetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	password, err := h.auth.ResetPassword(uint(id))
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	users, err := h.auth.ListUsers()
	if err != nil {
		fail(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users, "initialPassword": password})
}

func (h *Handler) changePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	if err := h.auth.ChangePassword(userID(c), req.CurrentPassword, req.NewPassword); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) createRoom(c *gin.Context) {
	var req struct {
		Password string                     `json:"password"`
		Settings services.RoomSettingsInput `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	snap, err := h.rooms.CreateRoom(userID(c), req.Password, req.Settings)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, snap)
}

func (h *Handler) joinRoom(c *gin.Context) {
	var req struct {
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	snap, err := h.rooms.JoinRoom(userID(c), req.Code, req.Password)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, snap)
}

func (h *Handler) room(c *gin.Context) {
	snap, err := h.rooms.Snapshot(c.Param("code"))
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, snap)
}

func (h *Handler) gameArchive(c *gin.Context) {
	archive, err := h.rooms.GameArchive(c.Param("code"))
	if err != nil {
		fail(c, http.StatusNotFound, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"game": archive})
}

func (h *Handler) ready(c *gin.Context) {
	var req struct {
		Ready bool `json:"ready"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	snap, err := h.rooms.SetReady(userID(c), c.Param("code"), req.Ready)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, snap)
}

func (h *Handler) settings(c *gin.Context) {
	var req services.RoomSettingsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	snap, err := h.rooms.UpdateSettings(userID(c), c.Param("code"), req)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, snap)
}

func (h *Handler) startRoll(c *gin.Context) {
	snap, err := h.rooms.StartRoll(userID(c), c.Param("code"))
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, snap)
}

func (h *Handler) roll(c *gin.Context) {
	snap, err := h.rooms.Roll(userID(c), c.Param("code"))
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, snap)
}

func (h *Handler) startGame(c *gin.Context) {
	snap, err := h.rooms.StartGame(userID(c), c.Param("code"))
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, snap)
}

func (h *Handler) contribution(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	snap, err := h.rooms.SubmitContribution(userID(c), c.Param("code"), req.Text)
	if err != nil {
		fail(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, snap)
}

func userID(c *gin.Context) uint {
	return c.MustGet("claims").(*auth.Claims).UserID
}

func fail(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"error": err.Error()})
}

func cors(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", cfg.FrontendURL)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
