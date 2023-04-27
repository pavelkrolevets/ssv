#   - mkdir ~/.kube/
#    - echo $STAGE_KUBECONFIG | base64 -d > kubeconfig
#    - mv kubeconfig ~/.kube/
#    - export KUBECONFIG=~/.kube/kubeconfig
#    - kubectl config get-contexts
#    - IMAGE_TAG=$CI_BUILD_REF
helm upgrade
      --install
      --namespace ssv
      --set image.tag=$IMAGE_TAG
      --values .k8/helm3/ssv-cluster-a/stage-values.yaml
      --wait
      ssv-cluster-a
      .k8/helm3/ssv-cluster-a/
    - helm upgrade
      --install
      --namespace ssv
      --set image.tag=$IMAGE_TAG
      --values .k8/helm3/ssv-cluster-b/stage-values.yaml
      --wait
      ssv-cluster-b
      .k8/helm3/ssv-cluster-b/
    - helm upgrade
      --install
      --namespace ssv
      --set image.tag=$IMAGE_TAG
      --values .k8/helm3/ssv-cluster-c/stage-values.yaml
      --wait
      ssv-cluster-c
      .k8/helm3/ssv-cluster-c/
    - helm upgrade
      --install
      --namespace ssv
      --set image.tag=$IMAGE_TAG
      --values .k8/helm3/ssv-cluster-d/stage-values.yaml
      --wait
      ssv-cluster-d
      .k8/helm3/ssv-cluster-d/
    - helm upgrade
      --install
      --namespace ssv
      --set image.tag=$IMAGE_TAG
      --values .k8/helm3/ssv-exporters/stage-values.yaml
      --wait
      ssv-exporters
      .k8/helm3/ssv-exporters/    