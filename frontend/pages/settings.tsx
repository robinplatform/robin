import Head from 'next/head';
import React from 'react';
import { useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';
import { useRpcMutation, useRpcQuery } from '../hooks/useRpcQuery';
import { z } from 'zod';
import { Settings } from '../components/Settings';

export default function RobinSettings() {
	const {
		data: config,
		isLoading,
		error,
	} = useRpcQuery({
		method: 'GetConfig',
		data: {},
		result: z.record(z.string(), z.unknown()),
	});

	const queryClient = useQueryClient();
	const { mutate: performUpdate, isLoading: isUpdating } = useRpcMutation({
		method: 'UpdateConfig',
		result: z.unknown(),
		onSuccess: () => {
			queryClient.invalidateQueries(['getConfig']);
			toast.success('Settings updated successfully');
		},
		onError: () => {
			toast.error('Failed to update settings');
		},
	});

	return (
		<div className={'full robin-pad'}>
			<Head>
				<title>Settings | Robin</title>
			</Head>

			<Settings
				title="Settings"
				schema={z.unknown()}
				isLoading={isLoading || isUpdating}
				error={error ? String(error) : undefined}
				value={config}
				onChange={(value) => {
					performUpdate(value);
				}}
			/>
		</div>
	);
}
