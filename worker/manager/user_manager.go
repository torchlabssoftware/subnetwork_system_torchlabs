package manager

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	util "github.com/snail007/goproxy/utils"
)

type User struct {
	ID          uuid.UUID
	Username    string
	Password    string
	Status      string
	IpWhitelist []string
	Pools       []PoolLimit
}

type CachedUser struct {
	User     *User
	CachedAt time.Time
	ExpireAt time.Time
}

type UserManager struct {
	cachedUsers        util.ConcurrentMap
	pendingValidations sync.Map
	//users              util.ConcurrentMap
	TTL time.Duration
	//onVerifyUser func(event Event)
}

func NewUserManager() *UserManager {
	userManager := &UserManager{
		cachedUsers: util.NewConcurrentMap(),
		//onVerifyUser: onVerifyUser,
		TTL: 24 * time.Hour,
	}
	go userManager.cleanupLoop()
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

func (u *UserManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
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

func (u *UserManager) VerifyUser(user, pass string) bool {
	respChan := make(chan bool)

	u.pendingValidations.Store(user, respChan)
	defer u.pendingValidations.Delete(user)

	_ = UserLoginPayload{
		Username: user,
		Password: pass,
	}

	//u.onVerifyUser(Event{Type: "verify_user", Payload: payload})

	select {
	case result := <-respChan:
		return result
	case <-time.After(5 * time.Second):
		log.Printf("[Captain] VerifyUser timeout for %s", user)
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

	// Parse pools with proper validation
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
		Status:      userPayload.Status,
		IpWhitelist: userPayload.IpWhitelist,
		Pools:       pools,
	}
	u.SetUser(user)

	// Signal successful auth
	respChan <- true
}
