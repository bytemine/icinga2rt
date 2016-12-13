# icinga2rt

icinga2rt is a tool which automatically creates, updates and closes request tracker tickets on status changes of
hosts or services monitored by icinga2.

## commandline arguments

	-config string
			configuration file (default "/etc/bytemine/icinga2rt.json")
	-debug
			debug mode, print log messages (default true)
	-debugevents
			print received events
	-example
			write example configuration file as icinga2rt.json.example to current directory
	-version
			display version and exit

## configuration

A configuration is expected to be in `/etc/bytemine/icinga2rt.json`, other paths can be used with the `-config` switch.
The `icinga2rt.json.example` file is a good starting point for a config. 

### explained example configuration

If parts of this are used, comments must be removed. Using the `-example` switch is recommended.

	{
		// Icinga2 API configuration
		"Icinga": {
			// URL of the Icinga2 API
			"URL": "https://monitoring.example.com:5665",
			// User and password for API access
			"User": "root",
			"Password": "secret",
			// Allow invalid SSL certificate chains (optional, if not specified defaults to false)
			"Insecure": true,
			// Try this many times to (re)connect before exiting, using exponential backoff (wait 1, 2, …, 2^n seconds)
			"Retries": 5
		},
		// Request Tracker API configuration
		"RT": {
			// URL of the RT API
			"URL": "https://support.example.com",
			// RT user and password
			"User": "apiuser",
			"Password": "secret",
			// Allow invalid SSL certificate chains (optional, if not specified defaults to false)
			"Insecure": true
		},
		// Event-ticket matching cache configuration
		"Cache": {
			// Path to the database file
			"File": "/var/lib/icinga2rt/icinga2rt.bolt"
		},
		// Ticket creation and updating configuration
		"Ticket": {
			// PermitFunction configures which events are acted upon
			// It is an array of { "State" : State, "StateType" : StateType } objects,
			// each defining a permitted combination on which an event is acted upon.
			// You should always define { "State": 0, "StateType": 1 } and
			// { "State": 0, "StateType": 0 } to allow OK events. The settings below
			// allow OK SOFT, OK HARD, WARNING HARD, CRITICAL HARD and UNKNOWN HARD 
			"PermitFunction": [
				{
					"State": 0,
					"StateType": 1
				},
				{
					"State": 0,
					"StateType": 0
				},
				{
					"State": 1,
					"StateType": 1
				},
				{
					"State": 2,
					"StateType": 1
				},
				{
					"State": 3,
					"StateType": 1
				}
			],
			// Nobody user to set as owner in RT
			"Nobody": "Nobody",
			// Queue to use for new tickets in RT
			"Queue": "general"
		}
	}

## running

### upstart

	description     "bytemine Icinga2 RT ticket creator"

	start on (net-device-up and local-filesystems and runlevel [2345])
	stop on runlevel [016]

	respawn
	respawn limit 10 5

	console log
	setuid icingatort
	setgid icingatort

	exec /usr/share/bytemine-icinga2rt/bytemine-icinga2rt

### systemd

I'm not really used to systemd, but this should work as a unit file:

	[Unit]
	Description=bytemine Icinga2 RT ticket creator
	After=network-online.target

	[Service]
	Restart=on-failure

	User=icingatort
	Group=icingatort

	ExecStart=/usr/share/bytemine-icinga2rt/bytemine-icinga2rt

	[Install]
	WantedBy=multi-user.target
