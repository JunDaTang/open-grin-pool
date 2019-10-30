import redis, datetime
stream = 'shares2'
client = redis.Redis(host='localhost', port=6379, db=0)

def get_minute_range(dt):
    dt_start = datetime.datetime(dt.year, dt.month, dt.day, dt.hour, dt.minute)
    dt_end = dt_start + datetime.timedelta(minutes=1)
    start = int(dt_start.timestamp() * 1000)
    end = int(dt_end.timestamp() * 1000 - 1)
    return [start, end]

now = datetime.datetime.now() - datetime.timedelta(minutes=1)

start, end = get_minute_range(now)
data = client.xrange(stream, start, end)

stats = {}
for i in data:
    d = i[1]
    user = str(d[b'user'], encoding = "utf-8")
    rig = str(d[b'rig'], encoding = "utf-8")
    stats[user] = {}
    stats[user]['total'] = 0
    stats[user][rig] = 0

for i in data:
    d = i[1]
    user = str(d[b'user'], encoding = "utf-8")
    rig = str(d[b'rig'], encoding = "utf-8")
    diff = int(d[b'diff'])
    stats[user]['total'] = stats[user]['total'] + diff
    stats[user][rig] = stats[user][rig] + diff
print(now)    
print(stats)
