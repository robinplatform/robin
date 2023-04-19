import { z } from 'zod';
import React from 'react';
import { runRpcQuery, useRpcQuery } from '../hooks/useRpcQuery';
import { useTopic } from '../../toolkit/react/stream';
import { ScrollWindow } from './ScrollWindow';
import toast from 'react-hot-toast';

type ProcessInfo = z.infer<typeof ProcessInfo>;
const ProcessInfo = z.object({});

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
export function ProcessDebugger() {
	const { data: processes = [], error } = useRpcQuery({
		method: 'ListProcesses',
		data: {},
		result: z.array(Process),
	});

	const [currentProcess, setCurrentProcess] = React.useState<Process>();
	const { state } = useTopic({
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
