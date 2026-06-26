package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"scam-directory/internal/models"
	"scam-directory/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	users       *repository.UserRepository
	oauthConfig *oauth2.Config
}

func NewAuthHandler(users *repository.UserRepository) *AuthHandler {
	cfg := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &AuthHandler{users: users, oauthConfig: cfg}
}

func makeJWT(userID, role string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func setAuthCookie(c *gin.Context, token string) {
	c.SetCookie("auth_token", token, int((30 * 24 * time.Hour).Seconds()), "/", "", os.Getenv("ENV") == "production", true)
}

// POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Name     string `json:"name"     binding:"required"`
		Email    string `json:"email"    binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, _ := h.users.GetByEmail(c, req.Email)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	hashStr := string(hash)

	user := &models.User{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: &hashStr,
		Role:         "user",
	}
	if err := h.users.Create(c, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	token, err := makeJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	setAuthCookie(c, token)
	c.JSON(http.StatusCreated, gin.H{"user": user, "token": token})
}

// POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"    binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByEmail(c, req.Email)
	if err != nil || user == nil || user.PasswordHash == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := makeJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	setAuthCookie(c, token)
	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

// POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.users.GetByID(c, userID.(string))
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// GET /auth/google
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	url := h.oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOnline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GET /auth/google/callback
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	oauthToken, err := h.oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "oauth exchange failed"})
		return
	}

	client := h.oauthConfig.Client(context.Background(), oauthToken)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info"})
		return
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode user info"})
		return
	}

	user, _ := h.users.GetByGoogleID(c, googleUser.ID)
	if user == nil {
		user, _ = h.users.GetByEmail(c, googleUser.Email)
	}

	if user == nil {
		user = &models.User{
			Email:     googleUser.Email,
			Name:      googleUser.Name,
			AvatarURL: &googleUser.AvatarURL,
			Role:      "user",
		}
		gid := googleUser.ID
		user.GoogleID = &gid
		if err := h.users.Create(c, user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
	} else if user.GoogleID == nil {
		h.users.UpdateGoogleID(c, user.ID, googleUser.ID, googleUser.AvatarURL)
	}

	token, err := makeJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	frontendURL := os.Getenv("CORS_ORIGIN")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	setAuthCookie(c, token)
	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/auth/callback?token="+token)
}
