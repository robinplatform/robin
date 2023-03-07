import Head from 'next/head';
import Router, { useRouter } from 'next/router';
import React from 'react';
import { AppWindow } from '../../../components/AppWindow';

export default function Page() {
	const router = useRouter();

	const id = typeof router.query.id === 'string' ? router.query.id : null;

	const [title, setTitle] = React.useState<string>(id ?? 'Loading');
	const [route, setRoute] = React.useState<string>('');

	React.useEffect(() => {
		if (!Router.isReady || id === null) {
			return;
		}

		const route = Router.asPath.substring('/app/'.length + (id ?? '').length);
		if (route === '') {
			setRoute('/');
		} else {
			setRoute(route);
		}
	}, [id, router.isReady]);

	React.useEffect(() => {
		if (id === null || route === '') {
			return;
		}

		console.log(
			'route changed, changing parent route:',
			route,
			Router.asPath.substring('/app/'.length + (id ?? '').length),
		);
		if (Router.asPath.substring('/app/'.length + (id ?? '').length) === route) {
			return;
		}

		Router.replace('/app/' + id + route, undefined, { shallow: true });
	}, [id, route]);

	return (
		<div className={'full col'}>
			<Head>
				<title>{`${title || 'Error'} | Robin`}</title>
			</Head>

			<div>{router.asPath}</div>
			<input
				type="text"
				value={route}
				onChange={(evt) => {
					console.log('changing route', evt.target.value);
					setRoute(evt.target.value);
				}}
			/>

			{id && !!route && (
				<AppWindow
					id={String(id)}
					setTitle={setTitle}
					route={route}
					setRoute={setRoute}
				/>
			)}
		</div>
	);
}
