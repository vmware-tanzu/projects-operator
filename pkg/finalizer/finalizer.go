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

package finalizer

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func AddFinalizer(obj v1.Object, finalizer string) {
	finalizers := obj.GetFinalizers()
	for _, f := range finalizers {
		if f == finalizer {
			return
		}
	}

	obj.SetFinalizers(append(finalizers, finalizer))
	return
}

func RemoveFinalizer(obj v1.Object, finalizer string) {
	finalizers := obj.GetFinalizers()

	newFinalizers := []string{}
	for _, f := range finalizers {
		if f != finalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}

	obj.SetFinalizers(newFinalizers)
	return
}
