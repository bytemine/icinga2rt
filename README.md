# icinga2rt

icinga2rt is a tool which automatically creates, updates and closes request tracker tickets on status changes of
hosts or services monitored by icinga2.

## Commandline Arguments

	-config string
		configuration file (default "/etc/bytemine/icinga2rt.json")
	-debug
		debug mode, print log messages
	-debugevents
		print received events
	-example
		write example configuration file as icinga2rt.json.example to current directory
	-exportCache string
		export contents of cache to this file, and quit
	-importCache string
		import contents of cache from this file, and quit
	-version
		display version and exit

## Configuration

A configuration is expected to be in `/etc/bytemine/icinga2rt.json`, other paths can be used with the `-config` switch.
The `icinga2rt.json.example` file is a good starting point for a config. 

### Explained Example Configuration

If parts of this are used, comments (//...) must be removed. Using the `-example` switch is recommended.

	{
		"Icinga": {
			"URL": "https://monitoring.example.com:5665", // URL to Icinga2 API
			"User": "root", // Icinga2 API user
			"Password": "secret", // Icinga2 API password
			"Insecure": true, // Ignore SSL certificate errors
			"Retries": 5 // Maximum tries for connecting to Icinga2 API
		},
		"RT": {
			"URL": "https://support.example.com", // Request Tracker base URL
			"User": "apiuser", // Request Tracker API user
			"Password": "secret", // Request Tracker password
			"Insecure": true // Ignore SSL certificate errors
		},
		"Cache": {
			"File": "/var/lib/icinga2rt/icinga2rt.bolt" // Path to cache file storing event-ticket associations
		},
		"Ticket": {
			"Mappings": [ // Mappings to decide the action for events. See mappings section further down.
				{
					"Condition": {
						"State": "OK",
						"Existing": false,
						"Owned": false
					},
					"Action": "ignore"
				},
				{
					"Condition": {
						"State": "OK",
						"Existing": true,
						"Owned": false
					},
					"Action": "delete"
				},
				{
					"Condition": {
						"State": "OK",
						"Existing": true,
						"Owned": true
					},
					"Action": "comment"
				},
				{
					"Condition": {
						"State": "WARNING",
						"Existing": false,
						"Owned": false
					},
					"Action": "create"
				},
				{
					"Condition": {
						"State": "WARNING",
						"Existing": true,
						"Owned": false
					},
					"Action": "comment"
				},
				{
					"Condition": {
						"State": "CRITICAL",
						"Existing": false,
						"Owned": false
					},
					"Action": "create"
				},
				{
					"Condition": {
						"State": "CRITICAL",
						"Existing": true,
						"Owned": false
					},
					"Action": "comment"
				},
				{
					"Condition": {
						"State": "UNKOWN",
						"Existing": false,
						"Owned": false
					},
					"Action": "create"
				},
				{
					"Condition": {
						"State": "UNKOWN",
						"Existing": true,
						"Owned": false
					},
					"Action": "comment"
				}
			],
			"Nobody": "Nobody", // A Request Tracker ticket is unowned if owned by this user.
			"Queue": "general", // Request Tracker queue where tickets are created
			"ClosedStatus": [ // List of Request Tracker stati for which tickets are considered to be closed.
				"done",
				"resolved",
				"deleted"
			]
		}
	}

### Mappings

A mapping sets the action if an event (and it's associated ticket) match a condition. A condition
consists of

- `State`: State of the Icinga2 event. String, one of `UNKNOWN`, `WARNING`, `CRITICAL`, `OK`. 
- `Existing`: If a Request Tracker ticket is existing for this event. Either `true` or `false`.
- `Owned`: If the Request Tracker ticket is owned by someone else than the nobody-user. String of the username.

The action can be one of 

- `create`: Create a new ticket for this event.
- `comment`: Comment an existing ticket for this event.
- `delete`: Delete an existing ticket.
- `ignore`: Ignore this event.

## Running

### Upstart

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
