package xsuportal

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/jmoiron/sqlx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/isucon/isucon10-final/webapp/golang/proto/xsuportal/resources"
)

const (
	WebpushVAPIDPrivateKeyPath = "../vapid_private.pem"
	WebpushSubject             = "xsuportal@example.com"
)

var priv *ecdsa.PrivateKey

func init() {
	var err error
	priv, err = GetVAPIDKey(WebpushVAPIDPrivateKeyPath)
	if err != nil {
		panic(err)
	}
}

func webPush(db sqlx.Ext, notificationPB *resources.Notification, contestantID string) error {
	subs, err := GetPushSubscriptions(db, contestantID)
	if err != nil {
		return fmt.Errorf("GetPushSubscriptions: %w", err)
	}

	for _, sub := range subs {
		if err := SendWebPush(priv, notificationPB, &sub); err != nil {
			return fmt.Errorf("SendWebPush: %w", err)
		}
	}

	return nil
}

type Notifier struct {
	mu      sync.Mutex
	options *webpush.Options
}

var (
	client = http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			ForceAttemptHTTP2: true,
		},
	}
	contestantsCache = sync.Map{}
	pushSubscriptionCache = sync.Map{}

)

func (n *Notifier) Reset() {
	contestantsCache = sync.Map{}
	pushSubscriptionCache = sync.Map{}
}

func (n *Notifier) VAPIDKey() *webpush.Options {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.options == nil {
		pemBytes, err := ioutil.ReadFile(WebpushVAPIDPrivateKeyPath)
		if err != nil {
			return nil
		}
		block, _ := pem.Decode(pemBytes)
		if block == nil {
			return nil
		}
		priKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil
		}
		priBytes := priKey.D.Bytes()
		pubBytes := elliptic.Marshal(priKey.Curve, priKey.X, priKey.Y)
		pri := base64.RawURLEncoding.EncodeToString(priBytes)
		pub := base64.RawURLEncoding.EncodeToString(pubBytes)
		n.options = &webpush.Options{
			Subscriber:      WebpushSubject,
			VAPIDPrivateKey: pri,
			VAPIDPublicKey:  pub,
			HTTPClient: &client,
		}
	}
	return n.options
}

func (n *Notifier) NotifyClarificationAnswered(db sqlx.Ext, c *Clarification, updated bool) error {
	var contestants []struct {
		ID     string `db:"id"`
		TeamID int64  `db:"team_id"`
	}
	if c.Disclosed.Valid && c.Disclosed.Bool {
		if val, ok := contestantsCache.Load(0); ok {
			contestants = val.([]struct{
				ID     string `db:"id"`
				TeamID int64  `db:"team_id"`
			})
		} else {
			err := sqlx.Select(
				db,
				&contestants,
				"SELECT `id`, `team_id` FROM `contestants` WHERE `team_id` IS NOT NULL",
			)
			if err != nil {
				return fmt.Errorf("select all contestants: %w", err)
			}
			contestantsCache.Store(0, contestants)
		}
	} else {
		if val, ok := contestantsCache.Load(c.TeamID); ok {
			contestants = val.([]struct{
				ID     string `db:"id"`
				TeamID int64  `db:"team_id"`
			})
		} else {
			err := sqlx.Select(
				db,
				&contestants,
				"SELECT `id`, `team_id` FROM `contestants` WHERE `team_id` = ?",
				c.TeamID,
			)
			if err != nil {
				return fmt.Errorf("select contestants(team_id=%v): %w", c.TeamID, err)
			}		
			contestantsCache.Store(c.TeamID, contestants)
		}
	}
	for _, contestant := range contestants {
		notificationPB := &resources.Notification{
			Content: &resources.Notification_ContentClarification{
				ContentClarification: &resources.Notification_ClarificationMessage{
					ClarificationId: c.ID,
					Owned:           c.TeamID == contestant.TeamID,
					Updated:         updated,
				},
			},
		}

		if n.VAPIDKey() != nil {
			notificationPB.Id = c.ID + 1000000
			notificationPB.CreatedAt = timestamppb.New(time.Now())
			go func() {
				if err := webPush(db, notificationPB, contestant.ID); err != nil {
					fmt.Printf("webPush: %v", err)
				}
			}()
		}
	}
	return nil
}

func (n *Notifier) NotifyBenchmarkJobFinished(db sqlx.Ext, job *BenchmarkJob) error {
	var contestants []struct {
		ID     string `db:"id"`
		TeamID int64  `db:"team_id"`
	}

	if val, ok := contestantsCache.Load(job.TeamID); ok {
		contestants = val.([]struct{
			ID     string `db:"id"`
			TeamID int64  `db:"team_id"`
		})
	} else {
		err := sqlx.Select(
			db,
			&contestants,
			"SELECT `id`, `team_id` FROM `contestants` WHERE `team_id` = ?",
			job.TeamID,
		)
		if err != nil {
			return fmt.Errorf("select contestants(team_id=%v): %w", job.TeamID, err)
		}		
		contestantsCache.Store(job.TeamID, contestants)
	}

	for _, contestant := range contestants {
		notificationPB := &resources.Notification{
			Content: &resources.Notification_ContentBenchmarkJob{
				ContentBenchmarkJob: &resources.Notification_BenchmarkJobMessage{
					BenchmarkJobId: job.ID,
				},
			},
		}

		if n.VAPIDKey() != nil {
			notificationPB.Id = job.ID
			notificationPB.CreatedAt = timestamppb.New(time.Now())
			go func() {
				if err := webPush(db, notificationPB, contestant.ID); err != nil {
					fmt.Printf("webPush: %v", err)
				}
			}()
		}
	}
	return nil
}

func GetVAPIDKey(path string) (*ecdsa.PrivateKey, error) {
	pemBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pem: %w", err)
	}
	for {
		block, rest := pem.Decode(pemBytes)
		pemBytes = rest
		if block == nil {
			break
		}
		ecPrivateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			continue
		}
		return ecPrivateKey, nil
	}
	return nil, fmt.Errorf("not found ec private key")
}

func MakeTestNotificationPB() *resources.Notification {
	return &resources.Notification{
		CreatedAt: timestamppb.New(time.Now().UTC()),
		Content: &resources.Notification_ContentTest{
			ContentTest: &resources.Notification_TestMessage{
				Something: rand.Int63n(10000),
			},
		},
	}
}

func GetPushSubscriptions(db sqlx.Queryer, contestantID string) ([]PushSubscription, error) {
	var subscriptions []PushSubscription
	if val, ok := pushSubscriptionCache.Load(contestantID); ok {
		subscriptions = val.([]PushSubscription)
	} else {
		err := sqlx.Select(
			db,
			&subscriptions,
			"SELECT * FROM `push_subscriptions` WHERE `contestant_id` = ?",
			contestantID,
		)
		if err != sql.ErrNoRows && err != nil {
			return nil, fmt.Errorf("select push subscriptions: %w", err)
		}
		pushSubscriptionCache.Store(contestantID, subscriptions)
	}
	return subscriptions, nil
}

func SendWebPush(vapidKey *ecdsa.PrivateKey, notificationPB *resources.Notification, pushSubscription *PushSubscription) error {
	b, err := proto.Marshal(notificationPB)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	message := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(message, b)

	vapidPrivateKey := base64.RawURLEncoding.EncodeToString(vapidKey.D.Bytes())
	vapidPublicKey := base64.RawURLEncoding.EncodeToString(elliptic.Marshal(vapidKey.Curve, vapidKey.X, vapidKey.Y))

	resp, err := webpush.SendNotification(
		message,
		&webpush.Subscription{
			Endpoint: pushSubscription.Endpoint,
			Keys: webpush.Keys{
				Auth:   pushSubscription.Auth,
				P256dh: pushSubscription.P256DH,
			},
		},
		&webpush.Options{
			Subscriber:      WebpushSubject,
			VAPIDPublicKey:  vapidPublicKey,
			VAPIDPrivateKey: vapidPrivateKey,
			HTTPClient: &client,
		},
	)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer resp.Body.Close()
	expired := resp.StatusCode == 410
	if expired {
		return fmt.Errorf("expired notification")
	}
	invalid := resp.StatusCode == 404
	if invalid {
		return fmt.Errorf("invalid notification")
	}
	return nil
}
