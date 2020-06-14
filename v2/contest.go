package contest

import (
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/sn/v2"
	"github.com/gin-gonic/gin"
)

const kind = "Contest"

type Client struct {
	*datastore.Client
}

func NewClient(dsClient *datastore.Client) Client {
	return Client{dsClient}
}

type Contests []*Contest
type Contest struct {
	c         *gin.Context
	Key       *datastore.Key `datastore:"__key__"`
	GameID    int64
	Type      sn.Type
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
	Type    sn.Type
	R       float64
	RD      float64
	Outcome float64
}

type Results []*Result
type ResultsMap map[*datastore.Key]Results
type Places []ResultsMap

func New(c *gin.Context, id int64, pk *datastore.Key, gid int64, t sn.Type, r, rd, outcome float64) *Contest {
	return &Contest{
		Key:     datastore.IDKey(kind, id, pk),
		GameID:  gid,
		Type:    t,
		R:       r,
		RD:      rd,
		Outcome: outcome,
	}
}

func GenContests(c *gin.Context, places Places) (cs Contests) {
	for _, rmap := range places {
		for ukey, rs := range rmap {
			for _, r := range rs {
				cs = append(cs, New(c, 0, ukey, r.GameID, r.Type, r.R, r.RD, r.Outcome))
			}
		}
	}
	return
}

func (client Client) UnappliedFor(c *gin.Context, ukey *datastore.Key, t sn.Type) (Contests, error) {
	q := datastore.NewQuery(kind).
		Ancestor(ukey).
		Filter("Applied=", false).
		Filter("Type=", int(t)).
		KeysOnly()

	ks, err := client.GetAll(c, q, nil)
	if err != nil {
		return nil, err
	}

	length := len(ks)
	if length == 0 {
		return nil, nil
	}

	cs := make(Contests, length)
	for i := range cs {
		cs[i] = new(Contest)
	}

	err = client.GetMulti(c, ks, cs)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

type ContestMap map[sn.Type]Contests

func (client Client) Unapplied(c *gin.Context, ukey *datastore.Key) (ContestMap, error) {
	q := datastore.NewQuery(kind).
		Ancestor(ukey).
		Filter("Applied=", false).
		KeysOnly()

	ks, err := client.GetAll(c, q, nil)
	if err != nil {
		return nil, err
	}

	length := len(ks)
	if length == 0 {
		return nil, nil
	}

	cs := make(Contests, length)
	for i := range cs {
		cs[i] = new(Contest)
	}

	err = client.GetMulti(c, ks, cs)
	if err != nil {
		return nil, err
	}

	cm := make(ContestMap, len(sn.Types))
	for _, c := range cs {
		c.Applied = true
		cm[c.Type] = append(cm[c.Type], c)
	}
	return cm, nil
}