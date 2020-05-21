This tool covers two goals
 * know when cardinality will explode 
 * make simple monitoring for prometheus, just check prometheus availability. 


service connect to $PROMETHEUS_URL/api/v1/status/tsdb and parse labelValueCountByLabelName seriesCountByMetricName and expose them viw :8080/metrics


If prometheus is not responsible for 5 times will create alert in alertmanager

ENV configs:
PROMETHEUS=http://prometheus-server.local:9090
ALERTMANAGER=http://alertmanager.local:9093
TIMEOUT=10
