import redis, datetime, requests
stream = 'shares2'
client = redis.Redis(host='localhost', port=6379, db=0, charset="utf-8", decode_responses=True)

def get_minute_range(dt, mins=1):
    dt_start = datetime.datetime(dt.year, dt.month, dt.day, dt.hour, dt.minute)
    dt_end = dt_start + datetime.timedelta(minutes=1)
    dt_start = dt_start - datetime.timedelta(minutes=mins)
    start = int(dt_start.timestamp() * 1000)
    end = int(dt_end.timestamp() * 1000 - 1)
    return [start, end]

def get_mins(dt):
    return dt.hour * 60 + dt.minute

now = datetime.datetime.now() - datetime.timedelta(minutes=1)

start, end = get_minute_range(now)
print(start)
print(end)
data = client.xrange(stream, start, end)
#print(data)

stats = {}
for i in data:
    d = i[1]
    user = d['user']
    rig = d['rig']
    if not stats.get(user):
        stats[user] = {}
        stats[user]['total'] = 0
    stats[user][rig] = 0
#print(stats)

for i in data:
    d = i[1]
    user = d['user']
    rig = d['rig']
    diff = int(d['diff'])
    stats[user]['total'] = stats[user]['total'] + diff
    stats[user][rig] = stats[user][rig] + diff
print(now)    
print(stats)

URL = ''
TOKEN = ''
data = {
    't': str(get_mins(now)),
    'stats': stats
}
payload = {
    'token': TOKEN,
    'data': data
}
resp = requests.post(URL, data=payload)
print(resp.status)