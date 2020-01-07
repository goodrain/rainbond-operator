#!/bin/bash
# openapi
mockgen -source=pkg/openapi/user/repositry.go -destination=pkg/openapi/user/mock/mock_repo.go -package=mock

# operator
mockgen -source=pkg/controller/rainbondcluster/pkg/install.go -destination=pkg/controller/rainbondcluster/pkg/mock/mock_install.go -package=mock