import sys
import json
import boto3


file_name = sys.argv[1]
table_name = sys.argv[2]

dynamodb = boto3.resource('dynamodb')

with open(file_name) as file:
	items = json.load(file)['Items']

	chunk = 25 # max items in batch write request
	for i in range(0, len(items), chunk):
		end = i + chunk
		if end > len(items):
			end = len(items)

		chunk_items = items[i:end]

		put_requests = []
		for chunk_item in chunk_items:
			item = {
				'id': chunk_item['id']['S'],
				'url': chunk_item['url']['S'],
				'title': chunk_item['title']['S'],
				'summary': chunk_item['summary']['S'],
			}

			put_requests.append({
				'PutRequest': {
					'Item': item
				}
			})

		request_items = {table_name: put_requests}

		response = dynamodb.batch_write_item(
			RequestItems=request_items,
			ReturnConsumedCapacity='NONE',
			ReturnItemCollectionMetrics='NONE',
		)

		if response['ResponseMetadata']['HTTPStatusCode'] != 200:
			print('response:', json.dumps(response, indent=4))
			exit(1)

print('successful batch upload')