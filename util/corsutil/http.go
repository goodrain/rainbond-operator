// WUTONG, Application Management Platform
// Copyright (C) 2014-2019 Wutong Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package corsutil

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

//SetCORS Enables cross-site script calls.
func SetCORS(ctx *gin.Context) {
	origin := ctx.GetHeader("Origin")
	ctx.Writer.Header().Add("Access-Control-Allow-Origin", origin)
	ctx.Writer.Header().Add("Access-Control-Allow-Methods", "POST,GET,OPTIONS,DELETE,PUT")
	ctx.Writer.Header().Add("Access-Control-Allow-Credentials", "true")
	ctx.Writer.Header().Add("Access-Control-Allow-Headers", "x-requested-with,content-type,Authorization")
}

//GetDomain get host domain
func GetDomain(ctx *gin.Context) string {
	domain := os.Getenv("DOMAIN")
	if domain == "" {
		domain = ctx.Request.Host
	}
	protocol := os.Getenv("DOMAIN_PROTOCOL")
	if protocol == "" {
		protocol = "http"
	}
	return fmt.Sprintf("%s://%s", protocol, domain)
}
