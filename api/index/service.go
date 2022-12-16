package index

import (
	"context"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/errors"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/weplanx/server/common"
	"github.com/weplanx/server/model"
	"github.com/weplanx/utils/captcha"
	"github.com/weplanx/utils/locker"
	"github.com/weplanx/utils/passlib"
	"github.com/weplanx/utils/passport"
	"github.com/weplanx/utils/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type Service struct {
	*common.Inject
	Passport        *passport.Passport
	Locker          *locker.Locker
	Captcha         *captcha.Captcha
	SessionsService *sessions.Service
}

func (x *Service) Login(ctx context.Context, email string, password string) (ts string, err error) {
	var user model.User
	if err = x.Db.Collection("users").
		FindOne(ctx, bson.M{
			"email":  email,
			"status": true,
		}).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			err = errors.NewPublic("the user does not exist or has been frozen")
			return
		}

		return
	}

	userId := user.ID.Hex()

	var maxLoginFailures bool
	if maxLoginFailures, err = x.Locker.Verify(ctx, userId, x.Values.LoginFailures); err != nil {
		return
	}
	if maxLoginFailures {
		err = errors.NewPublic("the user has exceeded the maximum number of login failures")
		return
	}

	var match bool
	if match, err = passlib.Verify(password, user.Password); err != nil {
		return
	}
	if !match {
		if err = x.Locker.Update(ctx, userId, x.Values.LoginTTL); err != nil {
			return
		}
		err = errors.NewPublic("the user email or password is incorrect")
		return
	}

	jti, _ := gonanoid.Nanoid()
	if ts, err = x.Passport.Create(userId, jti); err != nil {
		return
	}
	if err = x.Locker.Delete(ctx, userId); err != nil {
		return
	}
	if err = x.SessionsService.Set(ctx, userId, jti); err != nil {
		return
	}

	key := x.Values.Name("users", userId)
	if _, err = x.Redis.Del(ctx, key).Result(); err != nil {
		return
	}

	return
}

func (x *Service) Verify(ctx context.Context, ts string) (claims passport.Claims, err error) {
	if claims, err = x.Passport.Verify(ts); err != nil {
		return
	}
	var result bool
	if result, err = x.SessionsService.Verify(ctx, claims.UserId, claims.ID); err != nil {
		return
	}
	if !result {
		err = errors.NewPublic("the session token is inconsistent")
		return
	}

	// TODO: Check User Status

	if err = x.SessionsService.Renew(ctx, claims.UserId); err != nil {
		return
	}

	return
}

func (x *Service) GetRefreshCode(ctx context.Context, userId string) (code string, err error) {
	if code, err = gonanoid.Nanoid(); err != nil {
		return
	}
	if err = x.Captcha.Create(ctx, userId, code, 15*time.Second); err != nil {
		return
	}
	return
}

func (x *Service) RefreshToken(ctx context.Context, claims passport.Claims, code string) (ts string, err error) {
	if err = x.Captcha.Verify(ctx, claims.UserId, code); err != nil {
		return
	}
	if ts, err = x.Passport.Create(claims.UserId, claims.ID); err != nil {
		return
	}
	return
}

func (x *Service) Logout(ctx context.Context, userId string) (err error) {
	return x.SessionsService.Remove(ctx, userId)
}

func (x *Service) GetIdentity(ctx context.Context, userId string) (data model.User, err error) {
	key := x.Values.Name("users", userId)
	var exists int64
	if exists, err = x.Redis.Exists(ctx, key).Result(); err != nil {
		return
	}

	if exists == 0 {
		id, _ := primitive.ObjectIDFromHex(userId)
		option := options.FindOne().SetProjection(bson.M{"password": 0})
		if err = x.Db.Collection("users").
			FindOne(ctx, bson.M{
				"_id":    id,
				"status": true,
			}, option).
			Decode(&data); err != nil {
			return
		}

		var value string
		if value, err = sonic.MarshalString(data); err != nil {
			return
		}

		if err = x.Redis.Set(ctx, key, value, 0).Err(); err != nil {
			return
		}

		return
	}

	var result string
	if result, err = x.Redis.Get(ctx, key).Result(); err != nil {
		return
	}
	if err = sonic.UnmarshalString(result, &data); err != nil {
		return
	}

	return
}

func (x *Service) GetUser(ctx context.Context, userId string) (data map[string]interface{}, err error) {
	var user model.User
	if user, err = x.GetIdentity(ctx, userId); err != nil {
		return
	}

	data = map[string]interface{}{
		"email":  user.Email,
		"name":   user.Name,
		"avatar": user.Avatar,
	}

	return
}

func (x *Service) SetUser(ctx context.Context, userId string, data map[string]interface{}) (result interface{}, err error) {
	id, _ := primitive.ObjectIDFromHex(userId)
	update := bson.M{
		"$set": data,
	}
	if result, err = x.Db.Collection("users").
		UpdateByID(ctx, id, update); err != nil {
		return
	}

	key := x.Values.Name("users", userId)
	if _, err = x.Redis.Del(ctx, key).Result(); err != nil {
		return
	}

	return
}
