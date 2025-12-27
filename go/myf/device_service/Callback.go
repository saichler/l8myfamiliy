/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package device_service

import (
	"fmt"

	"github.com/saichler/l8myfamiliy/go/types/l8myfamily"
	"github.com/saichler/l8types/go/ifs"
)

type DeviceCallback struct{}

func (lc *DeviceCallback) Before(elem interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, bool, error) {
	if action == ifs.POST {
		device := elem.(*l8myfamily.Device)
		fmt.Println("[Device] ", device.Id, "-", device.FamilyId, "-", device.Name)
	}
	return nil, true, nil
}

func (lc *DeviceCallback) After(elem interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, bool, error) {
	return nil, true, nil
}
