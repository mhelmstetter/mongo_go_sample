# MongoDB Go Sample

- Simple go web application that exposes a few very basic HTTP GET endpoints (e.g. `/find` and `/upsert`) that will query and upsert MongoDB data
- Uses a really basic "dynamic configuration" by reading a config document from the database every 60 seconds
    - This can be used to dynamically change the context timeouts for MongoDB calls corresponding to each of the HTTP endpoints
- Stores basic metrics and events to a `metrics` collection for analysis, e.g. poolMonitor events and context timeouts, or other errors
- Includes configuration, build files, etc for deploying this app to AWS Elastic Beanstalk
- JMeter test scripts are included in the `jmeter` directory
