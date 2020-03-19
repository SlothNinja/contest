package contest

import (
	"time"

	"cloud.google.com/go/datastore"
	gtype "github.com/SlothNinja/type"
	"github.com/gin-gonic/gin"
)

const kind = "Contest"

type Contests []*Contest
type Contest struct {
	c *gin.Context
	// ID        int64          `gae:"$id"`
	// Parent    *datastore.Key `gae:"$parent"`
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

type Results []*Result
type ResultsMap map[*datastore.Key]Results
type Places []ResultsMap

func New(c *gin.Context, id int64, pk *datastore.Key, gid int64, t gtype.Type, r, rd, outcome float64) *Contest {
	return &Contest{
		Key: datastore.IDKey(kind, id, pk),
		// Parent:  pk,
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

//func (cs Contests) Keys() []*datastore.Key {
//	ks := make([]*datastore.Key, len(cs))
//	for i, c := range cs {
//		ks[i] = c.Key
//	}
//	return ks
//}

//func (cs Contests) Save(c *restful.Context) error {
//	keys := make([]*datastore.Key, len(cs))
//	now := time.Now()
//	for i, c := range cs {
//		keys[i] = c.Key
//		if c.CreatedAt.IsZero() {
//			c.CreatedAt = now
//		}
//		c.UpdatedAt = now
//	}
//	return datastore.RunInTransaction(ctx, func(tc appengine.Context) error {
//		_, err := gaelic.PutMulti(tc, keys, cs)
//		return err
//	}, nil)
//}

func UnappliedFor(c *gin.Context, ukey *datastore.Key, t gtype.Type) (Contests, error) {
	dsClient, err := datastore.NewClient(c, "")
	if err != nil {
		return nil, err
	}

	q := datastore.NewQuery(kind).
		Ancestor(ukey).
		Filter("Applied=", false).
		Filter("Type=", int(t)).
		KeysOnly()

	ks, err := dsClient.GetAll(c, q, nil)
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

	err = dsClient.GetMulti(c, ks, cs)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

type ContestMap map[gtype.Type]Contests

func Unapplied(c *gin.Context, ukey *datastore.Key) (ContestMap, error) {
	dsClient, err := datastore.NewClient(c, "")
	if err != nil {
		return nil, err
	}

	q := datastore.NewQuery(kind).
		Ancestor(ukey).
		Filter("Applied=", false).
		KeysOnly()

	ks, err := dsClient.GetAll(c, q, nil)
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
		// if ok := datastore.PopulateKey(cs[i], ks[i]); !ok {
		// 	return nil, fmt.Errorf("Unable to populate contest with key.")
		// }
	}

	err = dsClient.GetMulti(c, ks, cs)
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

//func (c *Contest) Save(ch chan<- datastore.Property) error {
//	// Time stamp
//	t := time.Now()
//	if c.CreatedAt.IsZero() {
//		c.CreatedAt = t
//	}
//	c.UpdatedAt = t
//	return datastore.SaveStruct(c, ch)
//}
//
//func (c *Contest) Load(ch <-chan datastore.Property) error {
//	return datastore.LoadStruct(c, ch)
//}
