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
	store store.Store
	q     queue.Client
	mux   *gin.Engine
}

// NewHandler creates an API handler. q may be nil; if provided, created task IDs
// will be published to the queue.
func NewHandler(s store.Store, q queue.Client) *Handler {
	r := gin.New()
	r.Use(gin.Recovery())
	h := &Handler{store: s, q: q, mux: r}
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
	is := h.store
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
		if err := is.CreateOrUpdateIdentity(&in); err != nil {
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
	is := h.store
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
		res, err = is.GetIdentityByZoomUserID(id)
	} else if typ == "teams" {
		res, err = is.GetIdentityByTeamsUserID(id)
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
	if c.Request.Method == "POST" {
		var in struct {
			Source  string            `json:"source"`
			Target  string            `json:"target"`
			Payload map[string]string `json:"payload"`
		}
		if err := c.BindJSON(&in); err != nil {
			c.String(400, "invalid json")
			return
		}
		t := &model.Task{Source: in.Source, Target: in.Target, Payload: in.Payload}
		id, err := h.store.CreateTask(t)
		if err != nil {
			c.String(500, "create error")
			return
		}
		// publish to queue if available
		if h.q != nil {
			if err := h.q.Publish(c.Request.Context(), id); err != nil {
				log.Printf("warning: failed to publish task %s to queue: %v", id, err)
			}
		}
		c.JSON(202, gin.H{"id": id})
		return
	}
	list, _ := h.store.ListTasks()
	c.JSON(200, list)
}

func (h *Handler) taskByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.String(400, "missing id")
		return
	}
	t, err := h.store.GetTask(id)
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
