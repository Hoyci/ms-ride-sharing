load('ext://restart_process', 'docker_build_with_restart')

k8s_yaml('./infra/development/k8s/base/app-config.yaml')
k8s_yaml('./infra/development/k8s/base/secrets.yaml')

### Postgres Instances (Database-per-Service) ###
k8s_yaml('./infra/development/k8s/base/postgres/user-db/deployment.yaml')
k8s_resource('user-service-db', port_forwards=['5433:5432'], labels="infra")

k8s_yaml('./infra/development/k8s/base/postgres/ride-db/deployment.yaml')
k8s_resource('ride-service-db', port_forwards=['5434:5432'], labels="infra")
### End of Postgres Instances ###

### Redis Service ###
k8s_yaml('./infra/development/k8s/base/redis/deployment.yaml')
k8s_resource('redis', port_forwards=['6379'], labels="infra")
### End of Redis Service ###

### RabbitMQ ###
k8s_yaml('./infra/development/k8s/base/rabbitmq/deployment.yaml')
k8s_resource('rabbitmq', port_forwards=['5672', '15672'], labels='infra')
### End RabbitMQ ###

### API Gateway ###
user_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-gateway ./services/api-gateway/cmd'

local_resource(
  'api-gateway-compile',
  user_compile_cmd,
  deps=['./services/api-gateway', './shared'], 
  labels="compiles",
)

docker_build_with_restart(
  'go-ride/api-gateway',
  '.',
  entrypoint=['/app/build/api-gateway'],
  dockerfile='./infra/development/docker/api-gateway.Dockerfile',
  only=[
    './build/api-gateway',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/services/api-gateway/deployment.yaml')
k8s_resource(
  'api-gateway', 
  resource_deps=['api-gateway-compile', 'rabbitmq'], 
  port_forwards=['8080:8080'],
  labels="services",
)
### End of user Service ###


### User Service ###
user_compile_cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/user-service ./services/user-service/cmd'

local_resource(
  'user-service-compile',
  user_compile_cmd,
  deps=['./services/user-service', './shared'], 
  labels="compiles",
)

docker_build_with_restart(
  'go-ride/user-service',
  '.',
  entrypoint=['/app/build/user-service'],
  dockerfile='./infra/development/docker/user-service.Dockerfile',
  only=[
    './build/user-service',
    './shared',
  ],
  live_update=[
    sync('./build', '/app/build'),
    sync('./shared', '/app/shared'),
  ],
)

k8s_yaml('./infra/development/k8s/services/user-service/deployment.yaml')
k8s_resource(
  'user-service', 
  resource_deps=['user-service-compile', 'rabbitmq'], 
  labels="services",
)
### End of user Service ###