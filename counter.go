package stored

import (
	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/directory"
)

// Counter allow you to operate different counters inside your object
type Counter struct {
	object *Object
	fields []*Field
	dir    directory.DirectorySubspace
}

func counterNew(ob *ObjectBuilder, fields []*Field) *Counter {
	ctr := Counter{
		object: ob.object,
		fields: fields,
	}
	ob.waitAll.Add(1)
	go func() {
		ob.waitInit.Wait()
		dir, err := ob.object.dir.CreateOrOpen(ob.object.db, []string{"counter"}, nil)
		if err != nil {
			panic("Object " + ob.object.name + " could not add counter directory")
		}
		ctr.dir = dir

		ob.mux.Lock()
		ob.object.counters[fieldsKey(fields)] = &ctr
		ob.mux.Unlock()
		ob.waitAll.Done()
	}()
	return &ctr
}

func (c *Counter) increment(tr fdb.Transaction, input *Struct) {
	t := input.getTuple(c.fields)
	tr.Add(c.dir.Pack(t), countInc)
}

func (c *Counter) decrement(tr fdb.Transaction, input *Struct) {
	t := input.getTuple(c.fields)
	tr.Add(c.dir.Pack(t), countDec)
}

// Get will get counter data
func (c *Counter) Get(data interface{}) *Promise {
	input := structAny(data)
	p := c.object.promiseInt64()
	p.doRead(func() Chain {
		t := input.getTuple(c.fields)
		incKey := c.dir.Pack(t)
		bytes, err := p.readTr.Get(incKey).Get()
		if err != nil {
			return p.fail(err)
		}
		if len(bytes) == 0 {
			// counter not created yet
			return p.done(int64(0))
			//return p.fail(ErrNotFound)
		}
		return p.done(ToInt64(bytes))
	})
	return p
}
