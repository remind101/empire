This is a [Hekad](https://hekad.readthedocs.org/en/v0.9.2/) container for forwarding logs to sumo, as well as librato via l2met.

# Development

1. Copy the `.env.logger.sample` and `.env.sumologic.sample` files and replace the values with actual values
2. Start the containers:
  
   ```console
   $ docker-compose up
   ```

This will run the empire-logger, the sumologic agent and a dummy container that will generate some log messages.
