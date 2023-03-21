import Head from 'next/head';
import React from 'react';
import { z } from 'zod';
import { useRpcQuery } from '../hooks/useRpcQuery';
import toast from 'react-hot-toast';
import { useStreamMethod } from '@robinplatform/toolkit/react/stream';

// This is a temporary bit of code to just display what's in the processes DB
// to make writing other features easier
function Processes() {
	const { data: processes, error } = useRpcQuery({
		method: 'ListProcesses',
		data: {},
		result: z.array(
			z.object({
				id: z.object({
					source: z.string(),
					key: z.string(),
				}),
				command: z.string(),
				args: z.array(z.string()),
			}),
		),
	});

	React.useEffect(() => {
		if (error) {
			toast.error(`${String(error)}`);
		}
	}, [error]);

	return (
		<div
			className={'full col robin-rounded robin-pad'}
			style={{ backgroundColor: 'DarkSlateGray', maxHeight: '100%' }}
		>
			<div>Processes</div>

			{/* The position relative/absolute stuff makes it so that the
			    inner div doesn't affect layout calculations of the surrounding div.
				I found this very confusing at first, so here's the SO post that I got it from:

				https://stackoverflow.com/questions/27433183/make-scrollable-div-take-up-remaining-height
			 */}
			<div className={'full'} style={{ position: 'relative', flexGrow: 1 }}>
				<div
					className={'full col robin-gap'}
					style={{
						position: 'absolute',
						top: 0,
						left: 0,
						right: 0,
						bottom: 0,
						overflowY: 'scroll',
					}}
				>
					{processes?.map((value) => {
						const key = `${value.id.source} ${value.id.key}`;
						return (
							<div
								key={key}
								className={'robin-rounded robin-pad'}
								style={{ backgroundColor: 'Coral' }}
							>
								{key}

								<pre>{JSON.stringify(value, null, 2)}</pre>
							</div>
						);
					})}
				</div>
			</div>
		</div>
	);
}

type TopicId = z.infer<typeof TopicId>;
const TopicId = z.object({
	category: z.string(),
	name: z.string(),
});

type TopicInfo = z.infer<typeof TopicInfo>;
const TopicInfo = z.object({
	id: TopicId,
	closed: z.boolean(),
	count: z.number(),
	subscriberCount: z.number(),
});

type MetaTopicInfo = z.infer<typeof MetaTopicInfo>;
const MetaTopicInfo = z.discriminatedUnion('kind', [
	z.object({
		kind: z.literal('update'),
		data: TopicInfo,
	}),
	z.object({
		kind: z.literal('close'),
		data: TopicId,
	}),
]);
function Topics() {
	const [selectedTopic, setSelectedTopic] = React.useState<
		TopicInfo & { key: string }
	>();

	const { data: initialTopics, error } = useRpcQuery({
		method: 'GetTopics',
		data: {},
		result: z.record(z.string(), TopicInfo),
		pathPrefix: '/api/apps/rpc',
	});

	const { state: topics } = useStreamMethod({
		methodName: 'SubscribeTopic',
		resultType: MetaTopicInfo,
		skip: !initialTopics,
		data: {
			id: {
				category: '@robin/topics',
				name: 'meta',
			},
		},
		initialState: initialTopics ?? {},
		reducer: (prev, packet) => {
			switch (packet.kind) {
				case 'update':
					return {
						...prev,
						[`${packet.data.id.category}/${packet.data.id.name}`]: packet.data,
					};
				case 'close':
					const a = { ...prev };
					// rome-ignore lint/performance/noDelete: I'm deleting a key from a record...
					delete a[`${packet.data.category}/${packet.data.name}`];

					// ...also the docs say this rule shouldn't even apply here. Like the rule is supposed to
					// ignore this case.
					return a;
			}
		},
	});

	const { state: topicMessages } = useStreamMethod<
		Record<string, string[]>,
		unknown
	>({
		methodName: 'SubscribeTopic',
		resultType: z.unknown(),
		skip: !selectedTopic?.id,
		data: {
			id: selectedTopic?.id,
		},
		initialState: {},
		reducer: (prev, packet) => {
			if (!selectedTopic) {
				return prev;
			}

			return {
				...prev,
				[selectedTopic.key]: [
					...(prev[selectedTopic.key] ?? []),
					JSON.stringify(packet),
				],
			};
		},
	});

	return (
		<div
			className={'full col robin-gap robin-rounded robin-pad'}
			style={{ backgroundColor: 'Black', maxHeight: '100%' }}
		>
			<div>Topics</div>

			{/* The position relative/absolute stuff makes it so that the
			    inner div doesn't affect layout calculations of the surrounding div.
				I found this very confusing at first, so here's the SO post that I got it from:

				https://stackoverflow.com/questions/27433183/make-scrollable-div-take-up-remaining-height
			 */}
			<div className={'full'} style={{ position: 'relative' }}>
				<div
					className={'full col robin-gap'}
					style={{
						position: 'absolute',
						top: 0,
						left: 0,
						right: 0,
						bottom: 0,
						overflowY: 'scroll',
					}}
				>
					{Object.entries(topics ?? {}).map(([key, topic]) => {
						return (
							<button
								key={key}
								className={'robin-rounded'}
								style={{
									backgroundColor: 'Coral',
									border:
										selectedTopic?.key === key
											? '3px solid blue'
											: '3px solid Coral',
								}}
								onClick={() => {
									setSelectedTopic((prevTopic) =>
										prevTopic?.key === key ? undefined : { ...topic, key },
									);
								}}
							>
								{key} with {topic.subscriberCount} subs
								{topic.closed ? '  X.X' : '  :)'}
							</button>
						);
					})}
				</div>
			</div>

			<div
				className={'full robin-rounded col robin-pad robin-gap'}
				style={{ backgroundColor: 'Brown' }}
			>
				{selectedTopic === undefined ? (
					<div>No topic is selected</div>
				) : (
					<>
						<div>
							Selected topic is{' '}
							{`${selectedTopic.id.category} - ${selectedTopic.id.name}`}
						</div>

						<div style={{ position: 'relative', flexGrow: 1 }}>
							<div
								className={'full col'}
								style={{
									position: 'absolute',
									top: 0,
									left: 0,
									right: 0,
									bottom: 0,
									overflowY: 'scroll',
								}}
							>
								{topicMessages[selectedTopic.key]?.map((msg, idx) => (
									<div key={`${msg} ${idx}`}>{msg}</div>
								))}
							</div>
						</div>
					</>
				)}
			</div>
		</div>
	);
}

export default function Home() {
	return (
		<div className={'robin-bg-light-blue robin-pad full'}>
			<Head>
				<title>Robin</title>
			</Head>

			<div
				className={
					'full col robin-rounded robin-pad robin-bg-dark-blue robin-gap'
				}
			>
				<div>Hello world!</div>

				<div className={'full robin-gap'} style={{ display: 'flex' }}>
					<div className={'full'} style={{ maxWidth: '30rem' }}>
						<Processes />
					</div>

					<div className={'full'} style={{ maxWidth: '30rem' }}>
						<Topics />
					</div>
				</div>
			</div>
		</div>
	);
}
