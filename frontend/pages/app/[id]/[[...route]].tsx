import Head from 'next/head';
import Router, { useRouter } from 'next/router';
import React from 'react';
import { AppWindow } from '../../../components/AppWindow';

export default function Page() {
	const router = useRouter();

	const id =
		router.isReady && typeof router.query.id === 'string'
			? router.query.id
			: null;

	const [title, setTitle] = React.useState<string>(id ?? 'Loading');
	const [route, setRoute] = React.useState<string>('');

	const urlRoute = router.isReady
		? router.asPath.substring('/app/'.length + (id ?? '').length)
		: null;

	console.log('route:', route, router.asPath);

	React.useEffect(() => {
		if (id === null) {
			return;
		}

		Router.push('/app/' + id + route, undefined, { shallow: true });
	}, [id, route]);

	return (
		<div className={'full col'}>
			<Head>
				<title>{`${title || 'Error'} | Robin`}</title>
			</Head>

			{id && urlRoute !== null && (
				<AppWindow
					id={String(id)}
					setTitle={setTitle}
					route={urlRoute}
					setRoute={setRoute}
				/>
			)}
		</div>
	);
}
