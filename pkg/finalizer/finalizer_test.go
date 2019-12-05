/*
Unauthorized use, copying or distribution of any source code in this
repository via any medium is strictly prohibited without the author's
express written consent.

ANY AUTHORIZED USE OF OR ACCESS TO THE SOFTWARE IS "AS IS", WITHOUT
WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO
THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT,TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package finalizer_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/pivotal/projects-operator/pkg/finalizer"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Finalizer", func() {
	DescribeTable("AddFinalizer",
		func(existingFinalizers []string, newFinalizer string, expectedFinalizers []string) {
			obj := &v1.ObjectMeta{}
			obj.SetFinalizers(existingFinalizers)

			AddFinalizer(obj, newFinalizer)
			Expect(obj.GetFinalizers()).To(Equal(expectedFinalizers))
		},
		Entry("no existing finalizers", []string{}, "finalizer.1", []string{"finalizer.1"}),
		Entry("one existing finalizer", []string{"finalizer.2"}, "finalizer.1", []string{"finalizer.2", "finalizer.1"}),
		Entry("some existing finalizers", []string{"finalizer.3", "finalizer.2"}, "finalizer.1", []string{"finalizer.3", "finalizer.2", "finalizer.1"}),
		Entry("finalizer exists", []string{"finalizer.1", "finalizer.2"}, "finalizer.1", []string{"finalizer.1", "finalizer.2"}),
	)

	DescribeTable("RemoveFinalizer",
		func(existingFinalizers []string, deleteFinalizer string, expectedFinalizers []string) {
			obj := &v1.ObjectMeta{}
			obj.SetFinalizers(existingFinalizers)

			RemoveFinalizer(obj, deleteFinalizer)
			Expect(obj.GetFinalizers()).To(Equal(expectedFinalizers))
		},
		Entry("no existing finalizers", []string{}, "finalizer.1", []string{}),
		Entry("one existing finalizer", []string{"finalizer.1"}, "finalizer.1", []string{}),
		Entry("some existing finalizers", []string{"finalizer.2", "finalizer.1"}, "finalizer.1", []string{"finalizer.2"}),
		Entry("no finalizer found", []string{"finalizer.3", "finalizer.2"}, "finalizer.1", []string{"finalizer.3", "finalizer.2"}),
	)
})
