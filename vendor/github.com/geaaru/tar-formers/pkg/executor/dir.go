/*
Copyright (C) 2021  Daniele Rondina <geaaru@sabayonlinux.org>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package executor

import (
	"os"
)

func (t *TarFormers) CreateDir(dir string, mode os.FileMode) (bool, error) {
	if t.Task.EnableMutex {
		mutex.Lock()
		defer mutex.Unlock()
	}

	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {

			return true, os.MkdirAll(dir, mode)
		} else {
			return false, err
		}
	}

	return false, nil
}
