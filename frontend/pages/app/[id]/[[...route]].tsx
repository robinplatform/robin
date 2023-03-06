import Head from 'next/head';
import { useRouter } from 'next/router';
import React from 'react';
import { AppWindow } from '../../../components/AppWindow';

export default function Page() {
	const router = useRouter();

	const id = typeof router.query.id === 'string' ? router.query.id : '';
	const [title, setTitle] = React.useState<string>(id ?? 'Loading');

	const { route } = router.query;

	const path = router.asPath.substring('/app/'.length + (id ?? '').length);

	return (
		<div className={'full col'}>
			<Head>
				<title>{title || 'Error'} | Robin</title>
			</Head>

			{id && <AppWindow id={String(id)} setTitle={setTitle} />}
		</div>
	);
}
