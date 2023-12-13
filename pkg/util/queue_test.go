package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func CaseQueue(t *testing.T) {
	q := NewQueue()
	Convey("Queue", t, func() {
		Convey("Enqueue", func() {
			q.Enqueue("test")
			q.Enqueue("test")
			i := q.Len()
			So(i, ShouldEqual, 2)
		})
		Convey("Dequeue", func() {
			dq := q.Dequeue()
			i := q.Len()
			So(i, ShouldEqual, 1)
			So(dq, ShouldEqual, "test")
		})
		Convey("Dequeue2", func() {
			_ = q.Dequeue()
			dq := q.Dequeue()
			So(dq, ShouldBeNil)
		})

	})
}

func TestQueue(t *testing.T) {
	CaseQueue(t)
}
