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
			"Filter": "", // Icinga2 event stream filter expression
			"LocalFilter": { // Local event filtering
				"Any": null,
				"All": [
					{
						"Users": [
							"request-tracker"
						]
					}
				]
			},
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
perform for this event. If the action is "status" two additional fields are required, one for the new
status of the ticket in Request Tracker and if the ticket-service mapping should be forgotten by icinga2rt.

 These mappings are stored in a CSV file with the columns

- state: one of `UNKNOWN`, `WARNING`, `CRITICAL`, `OK`
- old state: one of `UNKNOWN`, `WARNING`, `CRITICAL`, `OK` or an empty string for non existing tickets. 
- owned: one of `true` or `false`. should be `false` if old state is the empty string.
- action: one of `create`, `comment`, `delete`, `ignore`, `status`
- if action is `status`, two additional fields:
  - status: free text of the new Request Tracker status
  - invalidate: one of `true` or `false`. set to `true` if icinga2rt should forget about
    this ticket after processing this event so that the next event creates a new ticket.

The values supplied are read case-insensitive, but the values provided above are preferred.
Lines can be commented if their first character is `#`.

#### Example

	# state, old state, owned, action
	# ignore OK events if no old state is known
	OK,,false,ignore
	# set ticket status to resolved if unowned and was WARNING, CRITICAL
	OK,WARNING,false,status,resolved,true
	OK,CRITICAL,false,status,resolved,true
	# delete ticket if unowned and was UNKNOWN
	OK,UNKNOWN,false,delete
	# comment ticket if owned and was WARNING, CRITICAL or UNKNOWN
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

### Local Filters

Instead of using the Icinga2 filter (which aren't that well documented
:( ), you can filter the received events client side.  This happens by
giving either an `All` filter set or an `Any` filter set in LocalFilters.
These two sets are like conjunctive (cnf) and disjunctive (dnf) normal
forms. In the `All` set, every filter must match but only one field
of a filter has to match (cnf), the reverse logic applies to `Any`
sets. You can't specifiy both sets.  A filter set consists of a list
of event like objects, where each object can have the fields `Host`,
`Service`, `Users`, `Author`, `Text` and `NotificationType`. If a field
isn't present, it is ignored and has no impact on the filtering.
At the moment, only comparison for equality is possible.

#### Example: Filtering notifications by receiving users

We are only interested in creating tickets for two specific users `foo` and `bar`.
The easiest way to do this is to add a filter to the `All` set, consisting only of the
two users:

	"All": [
		{
			"Users": [
				"foo",
				"bar"
			]
		}
	]

Using the `Any` filter set, this would be:

	"Any": [
		{
			"Users": [
				"foo"
			]
		},
		{
			"Users": [
				"bar"
			]
		}
	]

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
