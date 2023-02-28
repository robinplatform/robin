import { useRouter } from 'next/router';
import { AppToolbar } from '../../../components/AppToolbar';
import { Settings } from '../../../components/Settings';
import { z } from 'zod';
import { useRpcMutation, useRpcQuery } from '../../../hooks/useRpcQuery';
import { Button } from '../../../components/Button';
import { ArrowLeftIcon } from '@primer/octicons-react';
import { toast } from 'react-hot-toast';
import { Alert } from '../../../components/Alert';
import { Spinner } from '../../../components/Spinner';
import { useQueryClient } from '@tanstack/react-query';
import Head from 'next/head';

export default function AppSettings() {
	const router = useRouter();
	const id = typeof router.query.id === 'string' ? router.query.id : null;

	const {
		data: appSettings,
		error: errLoadingAppSettings,
		isLoading,
	} = useRpcQuery({
		method: 'GetAppSettingsById',
		pathPrefix: '/api/apps/rpc',
		data: { appId: id },
		result: z.record(z.string(), z.unknown()),
	});

	const queryClient = useQueryClient();
	const { mutate: updateAppSettings } = useRpcMutation({
		method: 'UpdateAppSettings',
		pathPrefix: '/api/apps/rpc',
		result: z.record(z.string(), z.unknown()),

		onSuccess: () => {
			toast.success('Updated app settings');
			router.push(`/app/${id}`);
			queryClient.invalidateQueries(['GetAppSettingsById']);
		},
		onError: (err) => {
			toast.error(`Failed to update app settings: ${String(err)}`);
		},
	});

	if (!id) {
		return null;
	}
	return (
		<>
			<Head>
				<title>{id} Settings</title>
			</Head>

			<div className="full">
				<AppToolbar
					appId={id}
					actions={
						<>
							<Button
								size="sm"
								variant="primary"
								onClick={() => router.push(`/app/${id}`)}
							>
								<span style={{ marginRight: '.5rem' }}>
									<ArrowLeftIcon />
								</span>
								Back
							</Button>
						</>
					}
				/>

				<div
					className={'full'}
					style={{ padding: '0 .5rem', paddingBottom: '.5rem' }}
				>
					<>
						{isLoading && (
							<div
								className="full"
								style={{
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'center',
								}}
							>
								<p style={{ display: 'flex', alignItems: 'center' }}>
									<Spinner />
									<span style={{ marginLeft: '.5rem' }}>Loading...</span>
								</p>
							</div>
						)}
						{errLoadingAppSettings && (
							<Alert variant="error" title={'Failed to load app settings'}>
								{String(errLoadingAppSettings)}
							</Alert>
						)}
						{appSettings && (
							<Settings
								schema={z.unknown()}
								isLoading={false}
								error={undefined}
								value={appSettings}
								onChange={(value) =>
									updateAppSettings({
										appId: id,
										settings: value,
									})
								}
							/>
						)}
					</>
				</div>
			</div>
		</>
	);
}
