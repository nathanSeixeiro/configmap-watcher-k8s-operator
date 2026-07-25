# ConfigMapWatcher Operator

Operator Kubernetes em Go (Kubebuilder) para observar mudanças em um ConfigMap e enviar eventos HTTP para um endpoint externo.

## O que este operador faz

- Registra o CRD `ConfigMapWatcher` no grupo `apps.devops/v1alpha1`.
- Permite configurar qual ConfigMap observar por nome e namespace.
- Ao reconciliar, envia um payload JSON para o endpoint configurado em `spec.eventEndpoint`.

## API (`apps.devops/v1alpha1`)

### `ConfigMapWatcher.spec`

- `configMapName` (string, obrigatório): nome do ConfigMap monitorado.
- `configMapNamespace` (string, obrigatório): namespace do ConfigMap monitorado.
- `eventEndpoint` (string, obrigatório): URL HTTP para receber o evento.

### Exemplo de recurso

```yaml
apiVersion: apps.devops/v1alpha1
kind: ConfigMapWatcher
metadata:
    name: configmapwatcher-sample
    namespace: default
spec:
    configMapName: configmap-test
    configMapNamespace: default
    eventEndpoint: https://webhook.site/<seu-id>
```

## Pré-requisitos

- Go 1.24.6+
- Docker
- kubectl
- Um cluster Kubernetes (Kind recomendado para desenvolvimento)

## Desenvolvimento local com Kind

Criar cluster Kind para e2e (com imagem compatível com hosts cgroup v1):

```sh
make setup-test-e2e
```

Opcionalmente, em hosts cgroup v2, você pode sobrescrever a imagem do node:

```sh
make setup-test-e2e KIND_NODE_IMAGE=kindest/node:v1.36.1
```

## Fluxo de deploy

1. Gerar manifests e instalar CRD:

```sh
make install
```

2. Build e push da imagem do manager:

```sh
make docker-build docker-push IMG=<registry>/configmapwatcher:<tag>
```

3. Deploy do controller:

```sh
make deploy IMG=<registry>/configmapwatcher:<tag>
```

4. Aplicar recurso de exemplo:

```sh
kubectl apply -k config/samples/
```

## Teste rápido

1. Crie um ConfigMap de teste:

```sh
kubectl create configmap configmap-test --from-literal=key=valor -n default
```

2. Crie um `ConfigMapWatcher` apontando para esse ConfigMap.

3. Altere o ConfigMap:

```sh
kubectl patch configmap configmap-test -n default --type merge -p '{"data":{"key":"novo-valor"}}'
```

4. Verifique logs do controller:

```sh
kubectl logs -n configmapwatcher-system deploy/configmapwatcher-controller-manager -c manager
```

## Testes

- Unit tests:

```sh
make test
```

- End-to-end tests:

```sh
make test-e2e
```

## Troubleshooting

### CRD com domínio antigo (`apps.github.com`)

Se você mudou o domínio no projeto, mas o cluster ainda mostra o CRD antigo, rode:

```sh
make manifests
kubectl delete crd configmapwatchers.apps.github.com --ignore-not-found
make install
kubectl get crds | grep configmapwatchers.apps
```

O esperado é aparecer `configmapwatchers.apps.devops`.

### Warning `unrecognized format "int64"`

Esse warning pode aparecer no `controller-gen` e, neste projeto, não impede a criação do CRD.

## Limpeza

```sh
kubectl delete -k config/samples/ || true
make undeploy
make uninstall
make cleanup-test-e2e
```

## Comandos úteis

```sh
make help
```

## Referências

- Kubebuilder Book: https://book.kubebuilder.io/introduction.html

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0.
