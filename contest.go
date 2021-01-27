package contest

import (
	"errors"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/sn"
	gtype "github.com/SlothNinja/type"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

const (
	kind     = "Contest"
	msgEnter = "Entering"
	msgExit  = "Exiting"
)

var (
	ErrMissingKey   = errors.New("missing key")
	ErrNotFound     = errors.New("not found")
	ErrInvalidCache = errors.New("invalid cached value")
)

type Client struct {
	*sn.Client
}

func NewClient(dsClient *datastore.Client, logger *log.Logger, mcache *cache.Cache) *Client {
	return &Client{sn.NewClient(dsClient, logger, mcache, nil)}
}

type Contest struct {
	c         *gin.Context
	Key       *datastore.Key `datastore:"__key__"`
	GameID    int64
	Type      gtype.Type
	R         float64
	RD        float64
	Outcome   float64
	Applied   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Contest) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(c, ps)
}

func (c *Contest) Save() ([]datastore.Property, error) {
	c.UpdatedAt = time.Now()
	return datastore.SaveStruct(c)
}

func (c *Contest) LoadKey(k *datastore.Key) error {
	c.Key = k
	return nil
}

type Result struct {
	GameID  int64
	Type    gtype.Type
	R       float64
	RD      float64
	Outcome float64
}

type ResultsMap map[*datastore.Key][]*Result

func New(c *gin.Context, id int64, pk *datastore.Key, gid int64, t gtype.Type, r, rd, outcome float64) *Contest {
	return &Contest{
		Key:     datastore.IDKey(kind, id, pk),
		GameID:  gid,
		Type:    t,
		R:       r,
		RD:      rd,
		Outcome: outcome,
	}
}

func key(id int64, pk *datastore.Key) *datastore.Key {
	return datastore.IDKey(kind, id, pk)
}

func GenContests(c *gin.Context, places []ResultsMap) []*Contest {
	var cs []*Contest
	for _, rmap := range places {
		for ukey, rs := range rmap {
			for _, r := range rs {
				cs = append(cs, New(c, 0, ukey, r.GameID, r.Type, r.R, r.RD, r.Outcome))
			}
		}
	}
	return cs
}

func (client *Client) UnappliedFor(c *gin.Context, ukey *datastore.Key, t gtype.Type) ([]*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	q := datastore.NewQuery(kind).
		Ancestor(ukey).
		Filter("Applied=", false).
		Filter("Type=", int(t)).
		KeysOnly()

	ks, err := client.DS.GetAll(c, q, nil)
	if err != nil {
		return nil, err
	}

	length := len(ks)
	if length == 0 {
		return nil, nil
	}

	return client.getMulti(c, ks)
}

type ContestMap map[gtype.Type][]*Contest

func (client *Client) Unapplied(c *gin.Context, ukey *datastore.Key) (ContestMap, error) {
	q := datastore.NewQuery(kind).
		Ancestor(ukey).
		Filter("Applied=", false).
		KeysOnly()

	ks, err := client.DS.GetAll(c, q, nil)
	if err != nil {
		return nil, err
	}

	length := len(ks)
	if length == 0 {
		return nil, nil
	}

	cs, err := client.getMulti(c, ks)
	if err != nil {
		return nil, err
	}

	cm := make(ContestMap, len(gtype.Types))
	for _, c := range cs {
		c.Applied = true
		cm[c.Type] = append(cm[c.Type], c)
	}
	return cm, nil
}

func (client *Client) mcGet(c *gin.Context, k *datastore.Key) (*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	if k == nil {
		return nil, ErrMissingKey
	}

	ek := k.Encode()
	item, found := client.Cache.Get(ek)
	if !found {
		return nil, ErrNotFound
	}

	contest, ok := item.(*Contest)
	if !ok {
		return nil, ErrInvalidCache
	}
	return contest, nil
}

func (client *Client) dsGet(c *gin.Context, k *datastore.Key) (*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	if k == nil {
		return nil, ErrMissingKey
	}

	contest := new(Contest)
	err := client.DS.Get(c, k, contest)
	if err != nil {
		return nil, err
	}

	client.Cache.SetDefault(k.Encode(), contest)
	return contest, nil
}

func (client *Client) get(c *gin.Context, k *datastore.Key) (*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	contest, err := client.mcGet(c, k)
	if err != nil {
		return client.dsGet(c, k)
	}
	return contest, nil
}

func (client *Client) getMulti(c *gin.Context, ks []*datastore.Key) ([]*Contest, error) {
	client.Log.Debugf(msgEnter)
	defer client.Log.Debugf(msgExit)

	l, isNil := len(ks), true
	contests := make([]*Contest, l)
	me := make(datastore.MultiError, l)
	for i, k := range ks {
		contests[i], me[i] = client.get(c, k)
		if me[i] != nil {
			isNil = false
		}
	}
	if isNil {
		return contests, nil
	}
	return contests, me
}
