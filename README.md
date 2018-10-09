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
			"Mappings": "/etc/bytemine/icinga2rt.csv", // File with mappings
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

A mapping is the tuple of an events state, the old state (if any), if the ticket is owned, and an action to
perform for this event. These mappings are stored in a CSV file with the columns

- state: one of `UNKNOWN`, `WARNING`, `CRITICAL`, `OK`
- old state: one of `UNKNOWN`, `WARNING`, `CRITICAL`, `OK` or an empty string for non existing tickets. 
- owned: one of `true` or `false`. should be `false` if old state is the empty string.
- action: one of `create`, `comment`, `delete` or `ignore`

The values supplied are read case-insensitive, but the values provided above are preferred.
Lines can be commented if their first character is `#`.

#### Example

	# state, old state, owned, action
	# ignore OK events if no old state is known
	OK,,false,ignore
	# delete ticket if unowned and was WARNING, CRITICAL or UNKNOWN
	OK,WARNING,false,delete
	OK,CRITICAL,false,delete
	OK,UNKNOWN,false,delete
	# comment ticket if unowned and was WARNING, CRITICAL or UNKNOWN
	OK,WARNING,true,comment
	OK,CRITICAL,true,comment
	OK,UNKNOWN,true,comment
	# create tickets for WARNING, CRITICAL or UNKNOWN if not exisiting
	WARNING,,false,create
	CRITICAL,,false,create
	UNKNOWN,,false,create
	# ignore if state hasn't changed
	WARNING,WARNING,false,ignore
	WARNING,WARNING,true,ignore
	CRITICAL,CRITICAL,false,ignore
	CRITICAL,CRITICAL,true,ignore
	UNKNOWN,UNKNOWN,false,ignore
	UNKNOWN,UNKNOWN,true,ignore
	# comment tickets on state changes
	WARNING,CRITICAL,false,comment
	WARNING,CRITICAL,true,comment
	WARNING,UNKNOWN,false,comment
	WARNING,UNKNOWN,true,comment
	CRITICAL,WARNING,false,comment
	CRITICAL,WARNING,true,comment
	CRITICAL,UNKNOWN,false,comment
	CRITICAL,UNKNOWN,true,comment
	UNKNOWN,WARNING,false,comment
	UNKNOWN,WARNING,true,comment
	UNKNOWN,CRITICAL,false,comment
	UNKNOWN,CRITICAL,true,comment
	

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
