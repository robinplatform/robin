import Head from 'next/head';
import { useRouter } from 'next/router';
import React from 'react';
import { AppWindow } from '../../components/AppWindow';

export default function Page() {
	const router = useRouter();

	const id = `${router.query['id']}`;
	const [title, setTitle] = React.useState<string>('App | Robin');

	return (
		<div className={'full col'}>
			<Head>
				<title>{title}</title>
			</Head>

			<AppWindow id={id} setTitle={setTitle} />
		</div>
	);
}
