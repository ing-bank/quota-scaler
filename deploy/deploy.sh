echo "Creating namespaces"
oc new-project ichp-quota-scaler

echo "Deploying ichp-quota-scaler"
helm upgrade --install --force helm-quota-scaler ./helm-quota-scaler

echo "Done"
