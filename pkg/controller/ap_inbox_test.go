package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fsouza/slognil"
	vocab "github.com/go-ap/activitypub"
	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/repository"
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

func TestApInbox_CreateWorkoutActivity(t *testing.T) {
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

	remoteActorIRI := "https://wt-ap2.test/ap/users/runner"
	objectIRI := "https://wt-ap2.test/ap/users/runner/outbox/abc123#object"

	// Build a Create activity wrapping a WorkoutNote
	aepyCtx := `"aepy": "http://joinaepyornis.orh/ns#"`
	terms := `"workoutSport": "aepy:workoutSport", ` +
		`"workoutDuration": "aepy:workoutDuration", ` +
		`"workoutDistance": "aepy:workoutDistance", ` +
		`"workoutAverageSpeed": "aepy:workoutAverageSpeed", ` +
		`"workoutLocation": "aepy:workoutLocation", ` +
		`"workoutElevationGain": "aepy:workoutElevationGain"`
	payload := []byte(`{
		"@context": [
			"https://www.w3.org/ns/activitystreams",
			{` + aepyCtx + `, ` + terms + `}
		],
		"id": "https://wt-ap2.test/ap/users/runner/outbox/abc123",
		"type": "Create",
		"actor": "` + remoteActorIRI + `",
		"to": ["` + remoteActorIRI + `/followers"],
		"object": {
			"id": "` + objectIRI + `",
			"type": "Note",
			"attributedTo": "` + remoteActorIRI + `",
			"content": "Morning run\ndistance: 5.00 km",
			"published": "2025-06-15T08:30:00Z",
			"workoutSport": "running",
			"workoutDuration": 1800,
			"workoutDistance": 5000,
			"workoutAverageSpeed": 2.78,
			"workoutLocation": "Central Park",
			"workoutElevationGain": 50.5,
			"attachment": [
				{
					"type": "Image",
					"name": "route.png",
					"mediaType": "image/png",
					"url": "https://wt-ap2.test/images/route.png"
				}
			]
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
	c.Set(ap.RequestingActorContextKey, &ap.RequestActor{
		Actor: vocab.Actor{
			ID:   vocab.ID(remoteActorIRI),
			Name: vocab.DefaultNaturalLanguage("Test Runner"),
		},
	})

	err = ctrl.Inbox(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)

	// Verify the workout was created
	var workout model.Workout
	err = db.Where("external_object_iri = ?", objectIRI).First(&workout).Error
	require.NoError(t, err)

	assert.Equal(t, "Morning run", workout.Name)
	assert.Equal(t, model.WorkoutTypeRunning, workout.Type)
	assert.Equal(t, model.WorkoutVisibilityFollowers, workout.Visibility)
	assert.NotNil(t, workout.ActorIRI)
	assert.Equal(t, remoteActorIRI, *workout.ActorIRI)
	assert.NotNil(t, workout.ExternalObjectIRI)
	assert.Equal(t, objectIRI, *workout.ExternalObjectIRI)

	// Verify map data was set
	var mapData model.MapData
	err = db.Where("workout_id = ?", workout.ID).First(&mapData).Error
	require.NoError(t, err)
	assert.Equal(t, float64(5000), mapData.TotalDistance)
	assert.Equal(t, float64(50.5), mapData.TotalUp)
	assert.Equal(t, "Central Park", mapData.AddressString)

	// Verify image attachment with external URL was created
	var attachments []model.WorkoutAttachment
	err = db.Where("workout_id = ?", workout.ID).Find(&attachments).Error
	require.NoError(t, err)
	require.Len(t, attachments, 1)
	assert.Equal(t, "route.png", attachments[0].Filename)
	assert.NotNil(t, attachments[0].ExternalURL)
	assert.Equal(t, "https://wt-ap2.test/images/route.png", *attachments[0].ExternalURL)

	// Verify deduplication: sending the same activity again should not create a second workout
	req2 := httptest.NewRequest(http.MethodPost, "/ap/users/admin/inbox", bytes.NewReader(payload))
	req2.Header.Set(echo.HeaderContentType, ap.ContentType)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetPath("/ap/users/:username/inbox")
	c2.SetParamNames("username")
	c2.SetParamValues("admin")
	c2.Set(ap.RequestingActorContextKey, &ap.RequestActor{
		Actor: vocab.Actor{
			ID:   vocab.ID(remoteActorIRI),
			Name: vocab.DefaultNaturalLanguage("Test Runner"),
		},
	})

	err = ctrl.Inbox(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec2.Code)

	var count int64
	db.Model(&model.Workout{}).Where("external_object_iri = ?", objectIRI).Count(&count)
	assert.Equal(t, int64(1), count, "duplicate workout should not be created")
}

func TestApInbox_CreateWorkoutActivity_NotAReply(t *testing.T) {
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

	remoteActorIRI := "https://wt-ap2.test/ap/users/runner"

	// A reply note with inReplyTo should NOT be treated as a workout
	payload := []byte(`{
		"@context": [
			"https://www.w3.org/ns/activitystreams",
			{
				"aepy": "http://joinaepyornis.orh/ns#",
				"workoutSport": "aepy:workoutSport"
			}
		],
		"id": "https://wt-ap2.test/ap/users/runner/outbox/reply1",
		"type": "Create",
		"actor": "` + remoteActorIRI + `",
		"object": {
			"id": "https://wt-ap2.test/ap/users/runner/outbox/reply1#object",
			"type": "Note",
			"attributedTo": "` + remoteActorIRI + `",
			"content": "Nice run!",
			"inReplyTo": "https://wt-ap1.test/ap/users/admin/outbox/xyz#object",
			"workoutSport": "running"
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
	c.Set(ap.RequestingActorContextKey, &ap.RequestActor{
		Actor: vocab.Actor{
			ID:   vocab.ID(remoteActorIRI),
			Name: vocab.DefaultNaturalLanguage("Test Runner"),
		},
	})

	err = ctrl.Inbox(c)
	// This note has inReplyTo so it's a reply, not a workout.
	// It will try to resolve the reply target; if it doesn't find it,
	// the handler returns nil error but the workout should not be created.
	_ = err

	// Verify no workout was created
	var count int64
	db.Model(&model.Workout{}).Where("actor_iri IS NOT NULL").Count(&count)
	assert.Equal(t, int64(0), count, "reply should not create a workout")
}
