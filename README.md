This tool covers two goals
 * know when cardinality will explode 
 * make simple monitoring for prometheus, just check prometheus availability. 


service connect to $PROMETHEUS_URL/api/v1/status/tsdb and parse labelValueCountByLabelName seriesCountByMetricName
