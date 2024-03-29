import Head from 'next/head';
import React from 'react';
import { z } from 'zod';
import { runRpcQuery, useRpcQuery } from '../hooks/useRpcQuery';
import toast from 'react-hot-toast';
import { useTopicQuery } from '../../toolkit/react/stream';
import { ScrollWindow } from '../components/ScrollWindow';

type Process = z.infer<typeof Process>;
const Process = z.object({
	id: z.object({
		category: z.string(),
		key: z.string(),
	}),
	command: z.string(),
	args: z.array(z.string()),
});

// This is a temporary bit of code to just display what's in the processes DB
// to make writing other features easier
function Processes() {
	const { data: processes = [], error } = useRpcQuery({
		method: 'ListProcesses',
		data: {},
		result: z.array(Process),
	});

	const [currentProcess, setCurrentProcess] = React.useState<Process>();
	const { state } = useTopicQuery({
		topicId: currentProcess && {
			category: `/logs${currentProcess.id.category}`,
			key: currentProcess.id.key,
		},
		resultType: z.string(),
		fetchState: () =>
			runRpcQuery({
				method: 'GetProcessLogs',
				data: { processId: currentProcess?.id },
				result: z.object({
					counter: z.number(),
					text: z.string(),
				}),
			}).then(({ counter, text }) => ({ counter, state: text })),
		reducer: (prev, message) => {
			return prev + '\n' + message;
		},
	});

	React.useEffect(() => {
		if (error) {
			toast.error(`${String(error)}`);
		}
	}, [error]);

	return (
		<div
			className={'full col robin-rounded robin-gap robin-pad'}
			style={{ backgroundColor: 'DarkSlateGray', maxHeight: '100%' }}
		>
			<div>Processes</div>

			<ScrollWindow className={'full'} innerClassName={'col robin-gap'}>
				{processes?.map((value) => {
					const key = `${value.id.category} ${value.id.key}`;
					return (
						<div
							key={key}
							className={'robin-rounded robin-pad'}
							style={{ backgroundColor: 'Coral', width: '100%' }}
						>
							{key}

							<button onClick={() => setCurrentProcess(value)}>Select</button>

							<pre
								style={{
									width: '100%',
									whiteSpace: 'pre-wrap',
									wordWrap: 'break-word',
								}}
							>
								{JSON.stringify(value, null, 2)}
							</pre>
						</div>
					);
				})}
			</ScrollWindow>

			<ScrollWindow className={'full'} innerClassName={'col robin-gap'}>
				<pre
					style={{
						width: '100%',
						whiteSpace: 'pre-wrap',
						wordWrap: 'break-word',
					}}
				>
					{state}
				</pre>
			</ScrollWindow>
		</div>
	);
}

type TopicId = z.infer<typeof TopicId>;
const TopicId = z.object({
	category: z.string(),
	key: z.string(),
});

type TopicInfo = z.infer<typeof TopicInfo>;
const TopicInfo = z.object({
	id: TopicId,
	closed: z.boolean(),
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

	const { state: topics } = useTopicQuery({
		resultType: MetaTopicInfo,
		topicId: {
			category: '/topics',
			key: 'meta',
		},
		fetchState: () =>
			runRpcQuery({
				method: 'GetTopics',
				data: {},
				result: z.object({
					counter: z.number(),
					info: z.record(z.string(), TopicInfo),
				}),
				pathPrefix: '/api/apps/rpc',
			}).then(({ counter, info }) => ({ counter, state: info })),
		reducer: (prev, packet) => {
			switch (packet.kind) {
				case 'update':
					return {
						...prev,
						[`${packet.data.id.category}#${packet.data.id.key}`]: packet.data,
					};
				case 'close':
					const a: Record<string, TopicInfo> = { ...prev };
					// rome-ignore lint/performance/noDelete: I'm deleting a key from a record...
					delete a[`${packet.data.category}#${packet.data.key}`];

					// ...also the docs say this rule shouldn't even apply here. Like the rule is supposed to
					// ignore this case.
					return a;
			}
		},
	});

	const { state: topicMessages } = useTopicQuery<
		Record<string, string[]>,
		unknown
	>({
		resultType: z.unknown(),
		skip: !selectedTopic?.id,
		topicId: selectedTopic?.id,
		fetchState: async () => ({ state: {}, counter: 0 }),
		reducer: (prev, packet) => {
			if (!selectedTopic) {
				return prev;
			}

			const prevArray: string[] = prev[selectedTopic.key] ?? [];
			const message = JSON.stringify(packet);
			return {
				...prev,
				[selectedTopic.key]:
					prevArray.length > 20
						? [...prevArray.slice(1), message]
						: [...prevArray, message],
			};
		},
	});

	return (
		<div
			className={'full col robin-gap robin-rounded robin-pad'}
			style={{ backgroundColor: 'Black', maxHeight: '100%' }}
		>
			<div>Topics</div>

			<ScrollWindow className={'full'} innerClassName={'col robin-gap'}>
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
			</ScrollWindow>

			<div
				className={'full robin-rounded col'}
				style={{ backgroundColor: 'Brown' }}
			>
				<div className={'robin-pad'}>
					{selectedTopic === undefined ? (
						<div>No topic is selected</div>
					) : (
						<div>
							Selected topic is{' '}
							{`${selectedTopic.id.category} - ${selectedTopic.id.key}`}
						</div>
					)}
				</div>

				<ScrollWindow style={{ flexGrow: 1 }} innerClassName={'full col'}>
					{topicMessages?.[selectedTopic?.key ?? '']?.map((msg, idx) => (
						<div key={`${msg} ${idx}`} style={{ wordBreak: 'break-word' }}>
							{msg}
						</div>
					))}
				</ScrollWindow>
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
					<div className={'full'}>
						<Processes />
					</div>

					<div className={'full'}>
						<Topics />
					</div>
				</div>

				<div
					className={'robin-gap robin-pad robin-rounded'}
					style={{
						display: 'flex',
						backgroundColor: 'Gray',
					}}
				>
					<a href="/debug/pprof/" style={{ color: 'inherit' }}>
						Profiler endpoint
					</a>
				</div>
			</div>
		</div>
	);
}
