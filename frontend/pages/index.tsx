import Head from 'next/head';
import React from 'react';
import { z } from 'zod';
import { useRpcQuery } from '../hooks/useRpcQuery';
import toast from 'react-hot-toast';

function Processes() {
	const { data: processes, error } = useRpcQuery({
		method: 'ListProcesses',
		data: {},
		result: z.array(
			z.object({
				id: z.object({
					kind: z.string(),
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
						const key = `${value.id.kind} ${value.id.source} ${value.id.key}`;
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

				<div
					className={'full robin-gap'}
					style={{ display: 'flex', maxWidth: '30rem', maxHeight: '100%' }}
				>
					<Processes />
				</div>
			</div>
		</div>
	);
}
