#!/bin/bash
# openapi
mockgen -source=pkg/openapi/user/repositry.go -destination=pkg/openapi/user/mock/mock_repo.go -package=mock
mockgen -source=pkg/openapi/cluster/repository.go -destination=pkg/openapi/cluster/mock/mock_repo.go -package=mock
mockgen -source=pkg/openapi/cluster/usecase.go -destination=pkg/openapi/cluster/mock/mock_ucase.go -package=mock

# operator
mockgen -source=pkg/controller/rainbondvolume/plugin/plugin.go -destination=pkg/controller/rainbondvolume/plugin/mock/mock.go -package=mock