package manager

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	util "github.com/snail007/goproxy/utils"
)

type User struct {
	ID              uuid.UUID
	Username        string
	Password        string
	Status          string
	IpWhitelist     []string
	Pools           []PoolLimit
	Sessions        map[string]Upstream
	connectionCount int
}

type CachedUser struct {
	User     *User
	CachedAt time.Time
	ExpireAt time.Time
}

type UserManager struct {
	cachedUsers        util.ConcurrentMap
	pendingValidations sync.Map
	TTL                time.Duration
}

func NewUserManager() *UserManager {
	userManager := &UserManager{
		cachedUsers: util.NewConcurrentMap(),
		TTL:         1 * time.Hour,
	}
	go userManager.cleanupLoop(1 * time.Hour)
	go userManager.resetConnectionCount()
	return userManager
}

func (u *UserManager) SetUser(user *User) {
	u.cachedUsers.Set(user.Username, CachedUser{
		User:     user,
		CachedAt: time.Now(),
		ExpireAt: time.Now().Add(u.TTL),
	})
}

func (u *UserManager) GetUser(username string) (*User, bool) {
	if user, ok := u.cachedUsers.Get(username); ok {
		cachedUser := user.(CachedUser)
		if time.Now().Before(cachedUser.ExpireAt) {
			return cachedUser.User, true
		}
		u.RemoveUser(username)
	}
	return nil, false
}

func (u *UserManager) RemoveUser(username string) {
	u.cachedUsers.Remove(username)
}

func (u *UserManager) cleanupLoop(t time.Duration) {
	ticker := time.NewTicker(t)
	defer ticker.Stop()
	for range ticker.C {
		for item := range u.cachedUsers.IterBuffered() {
			cachedUser := item.Val.(CachedUser)
			if time.Now().After(cachedUser.ExpireAt) {
				u.RemoveUser(item.Key)
			}
		}
	}
}

func (u *UserManager) VerifyUser(username, password string, onVerifyUser func(event Event), pool string) bool {
	if user, ok := u.GetUser(username); ok {
		if user.Password == password && user.Status == "active" {
			for _, p := range user.Pools {
				if p.Tag == pool {
					if p.DataLimit > p.DataUsage {
						log.Printf("[UserManager] user login success [username:%v]/n", username)
						return true
					}
					log.Printf("[UserManager] user login failed [username:%v]/n", username)
					u.RemoveUser(username)
					return false
				}
			}
		}
		log.Printf("[UserManager] user login failed [username:%v]\n", username)
		u.RemoveUser(username)
		return false
	}
	respChan := make(chan bool)
	u.pendingValidations.Store(username, respChan)
	defer u.pendingValidations.Delete(username)
	payload := UserLoginPayload{
		Username: username,
		Password: password,
	}
	onVerifyUser(Event{Type: "verify_user", Payload: payload})
	select {
	case result := <-respChan:
		return result
	case <-time.After(5 * time.Second):
		log.Printf("[UserManager] VerifyUser timeout for %s", username)
		return false
	}
}

func (u *UserManager) processVerifyUserResponse(userPayload UserPayload) {
	ch, ok := u.pendingValidations.Load(userPayload.Username)
	if !ok {
		log.Printf("[UserManager] No pending validation for user: %s", userPayload.Username)
		return
	}
	respChan := ch.(chan bool)
	pools := make([]PoolLimit, 0)
	for _, pool := range userPayload.Pools {
		parts := strings.Split(pool, ":")
		if len(parts) < 3 {
			log.Printf("[UserManager] Invalid pool format (expected tag:limit:usage): %s", pool)
			continue
		}
		DataLimit, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("[UserManager] Invalid data limit in pool: %s", pool)
			continue
		}
		DataUsage, err := strconv.Atoi(parts[2])
		if err != nil {
			log.Printf("[UserManager] Invalid data usage in pool: %s", pool)
			continue
		}
		pools = append(pools, PoolLimit{
			Tag:       parts[0],
			DataLimit: DataLimit,
			DataUsage: DataUsage,
		})
	}
	user := &User{
		ID:          userPayload.ID,
		Username:    userPayload.Username,
		Password:    userPayload.Password,
		Status:      userPayload.Status,
		IpWhitelist: userPayload.IpWhitelist,
		Pools:       pools,
		Sessions:    make(map[string]Upstream),
	}
	u.SetUser(user)
	respChan <- true
}

func (u *UserManager) addConnection(username string) error {
	if user, ok := u.cachedUsers.Get(username); ok {
		cachedUser := user.(CachedUser)
		if cachedUser.User.connectionCount < 50 {
			cachedUser.User.connectionCount++
			return nil
		}
		return fmt.Errorf("user %s has reached the maximum number of connections", username)
	}
	return fmt.Errorf("user %s not found", username)
}

func (u *UserManager) removeConnection(username string) {
	if user, ok := u.cachedUsers.Get(username); ok {
		cachedUser := user.(CachedUser)
		if cachedUser.User.connectionCount >= 0 {
			cachedUser.User.connectionCount--
		}
		return
	}
}

func (u *UserManager) resetConnectionCount() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		for item, val := range u.cachedUsers.Items() {
			cachedUser := val.(CachedUser)
			cachedUser.User.connectionCount = 0
			u.cachedUsers.Set(item, cachedUser)
		}

	}
}
