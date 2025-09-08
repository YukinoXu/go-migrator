package api

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"

	"example.com/go-migrator/internal/model"
	"example.com/go-migrator/internal/queue"
	"example.com/go-migrator/internal/store"
)

type Handler struct {
	stm *store.StoreManager
	q   queue.Client
	mux *gin.Engine
}

// NewHandler creates an API handler. q may be nil; if provided, created task IDs
// will be published to the queue.
func NewHandler(s *store.StoreManager, q queue.Client) *Handler {
	r := gin.New()
	r.Use(gin.Recovery())
	h := &Handler{stm: s, q: q, mux: r}
	h.routes()
	return h
}

// Router returns the underlying http.Handler (gin engine implements http.Handler)
func (h *Handler) Router() *gin.Engine { return h.mux }

func (h *Handler) routes() {
	// tasks
	h.mux.POST("/tasks", h.tasks)
	h.mux.GET("/tasks", h.tasks)
	h.mux.GET("/tasks/:id", h.taskByID)

	// identities
	h.mux.POST("/identities", h.identities)
	h.mux.GET("/identities", h.identities)
	h.mux.GET("/identities/zoom/:id", h.identityByKey)
	h.mux.GET("/identities/teams/:id", h.identityByKey)
}

// identities handles POST to create/update and GET to list or query identities.
func (h *Handler) identities(c *gin.Context) {
	is := h.stm.Identity
	if c.Request.Method == "POST" {
		var in model.Identity
		if err := c.BindJSON(&in); err != nil {
			c.String(400, "invalid json")
			return
		}
		if in.ZoomUserID == "" {
			c.String(400, "zoom_user_id required")
			return
		}
		if err := is.Create(&in); err != nil {
			log.Printf("identity store error: %v", err)
			c.String(500, "internal")
			return
		}
		c.Status(204)
		return
	}
	// GET not implemented
	c.String(501, "not implemented")
}

// identityByKey supports GET /identities/zoom/{zoomUserID} and /identities/teams/{teamsUserID}
func (h *Handler) identityByKey(c *gin.Context) {
	is := h.stm.Identity
	// route contains either zoom/:id or teams/:id
	typ := ""
	if strings.HasPrefix(c.FullPath(), "/identities/zoom/") {
		typ = "zoom"
	} else if strings.HasPrefix(c.FullPath(), "/identities/teams/") {
		typ = "teams"
	}
	id := c.Param("id")
	if id == "" {
		c.String(400, "missing id")
		return
	}
	var (
		res *model.Identity
		err error
	)
	if typ == "zoom" {
		res, err = is.GetByZoomID(id)
	} else if typ == "teams" {
		res, err = is.GetByTeamsID(id)
	} else {
		c.String(400, "invalid identity type")
		return
	}
	if err != nil {
		if err == store.ErrNotFound {
			c.String(404, "not found")
			return
		}
		log.Printf("identity lookup error: %v", err)
		c.String(500, "internal")
		return
	}
	c.JSON(200, res)
}

func (h *Handler) tasks(c *gin.Context) {
	ts := h.stm.Task
	if c.Request.Method == "POST" {
		var in model.Task
		if err := c.BindJSON(&in); err != nil {
			c.String(400, "invalid json")
			return
		}
		if err := ts.Create(&in); err != nil {
			log.Printf("task store error: %v", err)
			c.String(500, "internal")
			return
		}
		c.Status(204)
		return
	}
	projectID := c.Param("projectId")
	status := c.Param("status")
	list, err := ts.ListByProject(projectID, status)
	if err != nil {
		log.Printf("task store error: %v", err)
		c.String(500, "internal")
		return
	}
	c.JSON(200, list)
}

func (h *Handler) taskByID(c *gin.Context) {
	ts := h.stm.Task
	id := c.Param("id")
	if id == "" {
		c.String(400, "missing id")
		return
	}
	t, err := ts.GetByID(id)
	if err != nil {
		if err == store.ErrNotFound {
			c.String(404, "not found")
			return
		}
		log.Printf("store error: %v", err)
		c.String(500, "internal")
		return
	}
	c.JSON(200, t)
}
