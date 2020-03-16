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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectAccessSpec defines the desired state of ProjectAccess
type ProjectAccessSpec struct {
}

// ProjectAccessStatus defines the observed state of ProjectAccess
type ProjectAccessStatus struct {
	Projects []string `json:"projects,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// ProjectAccess is the Schema for the projectaccesses API
type ProjectAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectAccessSpec   `json:"spec,omitempty"`
	Status ProjectAccessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectAccessList contains a list of ProjectAccess
type ProjectAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectAccess{}, &ProjectAccessList{})
}
