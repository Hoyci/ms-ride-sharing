load('ext://restart_process', 'docker_build_with_restart')

k8s_yaml('./k8s/development/base/app-config.yaml')
k8s_yaml('./k8s/development/base/secrets.yaml')

### Postgres Instances (Database-per-Service) ###
k8s_yaml('./k8s/development/base/postgres/user-db/deployment.yaml')
k8s_resource('user-service-db', port_forwards=['5433:5432'], labels="infra")

k8s_yaml('./k8s/development/base/postgres/ride-db/deployment.yaml')
k8s_resource('ride-service-db', port_forwards=['5434:5432'], labels="infra")
### End of Postgres Instances ###

### Redis Service ###
k8s_yaml('./k8s/development/base/redis/deployment.yaml')
k8s_resource('redis', port_forwards=['6379'], labels="infra")
### End of Redis Service ###

### RabbitMQ ###
k8s_yaml('./k8s/development/base/rabbitmq/deployment.yaml')
k8s_resource('rabbitmq', port_forwards=['5672', '15672'], labels='infra')
### End RabbitMQ ###
