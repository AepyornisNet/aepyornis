package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	ap "github.com/AepyornisNet/aepyornis/pkg/activitypub"
	"github.com/AepyornisNet/aepyornis/pkg/container"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/fsouza/slognil"
	vocab "github.com/go-ap/activitypub"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApInbox_AcceptFollowActivity(t *testing.T) {
	db, err := model.Connect("memory", "", false, slognil.NewLogger())
	require.NoError(t, err)

	repos := repository.New(db)
	ctr := container.NewContainer(db, nil, nil, nil, slognil.NewLogger(), nil, repos)
	ctrl := NewApInboxController(ctr)

	localUser := &model.User{
		UserData: model.UserData{
			Username:    "admin",
			Name:        "Admin",
			Active:      true,
			ActivityPub: true,
		},
		UserSecrets: model.UserSecrets{
			Password: "pass",
		},
	}
	localUser.SetDB(db)
	require.NoError(t, localUser.Create(db))

	remoteActorIRI := "https://wt-ap2.test/ap/users/admin"
	_, err = repos.Follower.UpsertFollowingRequest(localUser.ID, remoteActorIRI, remoteActorIRI+"/inbox")
	require.NoError(t, err)

	payload := []byte(`{
		"@context":"https://www.w3.org/ns/activitystreams",
		"id":"https://wt-ap2.test/ap/users/admin#accept-follow-1",
		"type":"Accept",
		"actor":"https://wt-ap2.test/ap/users/admin",
		"object":{
			"type":"Follow",
			"actor":"https://wt-ap1.test/ap/users/admin",
			"object":"https://wt-ap2.test/ap/users/admin"
		}
	}`)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/ap/users/admin/inbox", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, ap.ContentType)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/ap/users/:username/inbox")
	c.SetParamNames("username")
	c.SetParamValues("admin")
	c.Set(ap.RequestingActorContextKey, &ap.RequestActor{Actor: vocab.Actor{ID: vocab.ID(remoteActorIRI)}})

	err = ctrl.Inbox(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)

	approved, err := repos.Follower.IsFollowingApprovedByActorIRI(localUser.ID, remoteActorIRI)
	require.NoError(t, err)
	assert.True(t, approved)
}

func TestApInbox_CreateRemoteWorkoutActivity(t *testing.T) {
	db, err := model.Connect("memory", "", false, slognil.NewLogger())
	require.NoError(t, err)

	repos := repository.New(db)
	ctr := container.NewContainer(db, nil, nil, nil, slognil.NewLogger(), nil, repos)
	ctrl := NewApInboxController(ctr)

	localUser := &model.User{
		UserData: model.UserData{
			Username:    "admin",
			Name:        "Admin",
			Active:      true,
			ActivityPub: true,
		},
		UserSecrets: model.UserSecrets{
			Password: "pass",
		},
	}
	localUser.SetDB(db)
	require.NoError(t, localUser.Create(db))

	remoteActorIRI := "https://wt-ap2.test/ap/users/admin"
	payload := []byte(`{
		"@context":"https://www.w3.org/ns/activitystreams",
		"id":"https://wt-ap2.test/ap/users/admin#create-workout-1",
		"type":"Create",
		"actor":"https://wt-ap2.test/ap/users/admin",
		"object":{
			"id":"https://wt-ap2.test/ap/users/admin/outbox/abc#object",
			"type":"Note",
			"content":"Morning run",
			"workoutFitFile":"https://wt-ap2.test/ap/users/admin/outbox/abc/fit"
		}
	}`)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/ap/users/admin/inbox", bytes.NewReader(payload))
	req.Header.Set(echo.HeaderContentType, ap.ContentType)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/ap/users/:username/inbox")
	c.SetParamNames("username")
	c.SetParamValues("admin")
	c.Set(ap.RequestingActorContextKey, &ap.RequestActor{Actor: vocab.Actor{ID: vocab.ID(remoteActorIRI)}})

	err = ctrl.Inbox(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)

	status := &model.APStatus{}
	err = db.Where("object_id = ?", "https://wt-ap2.test/ap/users/admin/outbox/abc#object").First(status).Error
	require.NoError(t, err)
	assert.Equal(t, model.APStatusTypeWorkout, status.StatusType)
	assert.Equal(t, "remote", status.Origin)
	if assert.NotNil(t, status.ActorIRI) {
		assert.Equal(t, remoteActorIRI, *status.ActorIRI)
	}
}
