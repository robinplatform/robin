import Head from 'next/head';
import { useRouter } from 'next/router';
import React from 'react';
import { AppWindow } from '../../../components/AppWindow';

export default function Page() {
	const router = useRouter();

	const id = typeof router.query.id === 'string' ? router.query.id : null;
	const [title, setTitle] = React.useState<string>(id ?? 'Loading');

	return (
		<div className={'full col'}>
			<Head>
				<title>{`${title || 'Error'} | Robin`}</title>
			</Head>

			{id && <AppWindow id={String(id)} setTitle={setTitle} />}
		</div>
	);
}
