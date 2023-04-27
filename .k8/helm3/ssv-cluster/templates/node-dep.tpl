{{- define "node-dep" }}
---
apiVersion: v1
kind: Service
metadata:
  name: node-{{ $clustername }}-{{ .node_name }}-svc
  namespace: "ns"
  labels:
    app: node-{{ $clustername }}-{{ .node_name }}
spec:
  type: ClusterIP
  ports:
    - port: 12001
      protocol: UDP
      targetPort: 12001
      name: port-12001
    - port: 13001
      protocol: TCP
      targetPort: 13001
      name: port-13001
    - port: 15001
      protocol: TCP
      targetPort: 15001
      name: port-15001
  selector:
    app: node-{{ $clustername }}-{{ .node_name }}
---
{{- end }}