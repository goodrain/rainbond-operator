// Copyright (C) 2014-2018 Goodrain Co., Ltd.
// RAINBOND, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package uuidutil

import (
	"crypto/sha256"
	"encoding/hex"
)

// NewStableUUID 基于输入字符串生成稳定的32位uuid
// 相同的输入会生成相同的输出，用于确保同一集群多次安装生成相同的EID
func NewStableUUID(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:32]
}
