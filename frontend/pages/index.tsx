import Head from 'next/head';
import React from 'react';
import { z } from 'zod';
import { useRpcQuery } from '../hooks/useRpcQuery';
import toast from 'react-hot-toast';
import { Stream } from '@robinplatform/toolkit/stream';

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
function Topics() {
	const [selectedTopic, setSelectedTopic] = React.useState<TopicId>();
	const { data: topics, error } = useRpcQuery({
		method: 'GetTopics',
		data: {},
		result: z.array(TopicId),
		pathPrefix: '/api/apps/rpc',
	});

	React.useEffect(() => {
		const id = `${Math.random()} adsf`;
		const runner = async () => {
			const stream = await Stream.callStreamRpc('SubscribeTopic', id);
			stream.onmessage = (data) => {
				console.log('subscribe-message', data);
			};
			stream.onerror = (err) => {
				console.log('error', err);
			};
			stream.start({
				id: {
					category: 'hello',
					name: 'blahlahs',
				},
			});
		};
		runner();
	}, []);

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
					{topics?.map((id) => {
						const key = `${id.category}-${id.name}`;
						return (
							<button
								key={key}
								className={'robin-rounded robin-pad'}
								onClick={() => {
									setSelectedTopic(id);
								}}
								style={{ backgroundColor: 'Coral' }}
							>
								{key}
							</button>
						);
					})}
				</div>
			</div>

			<div
				className={'full robin-rounded robin-pad'}
				style={{ backgroundColor: 'Brown' }}
			>
				{selectedTopic === undefined ? (
					<div>No topic is selected</div>
				) : (
					<>
						<div>
							Selected topic is{' '}
							{`${selectedTopic.category}-${selectedTopic.name}`}
						</div>

						<div>Category: {selectedTopic.category}</div>
						<div>Name: {selectedTopic.name}</div>
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
