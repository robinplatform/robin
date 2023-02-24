import { useRouter } from 'next/router';
import React from 'react';
import { Alert } from './Alert';
import { Button } from './Button';

type Props = {
	id: string | undefined;
	setTitle: React.Dispatch<React.SetStateAction<string>>;
};

function AppWindowContent({ id, setTitle }: Props) {
	const router = useRouter();

	const iframeRef = React.useRef<HTMLIFrameElement | null>(null);
	const [error, setError] = React.useState<string | null>(null);

	React.useEffect(() => {
		const onMessage = (message: MessageEvent) => {
			try {
				switch (message.data.type) {
					case 'locationUpdate': {
						const location = {
							pathname: window.location.pathname,
							search: new URL(message.data.location).search,
						};
						router.push(location, undefined, { shallow: true });
						break;
					}

					case 'titleUpdate':
						setTitle(title => message.data.title || title);
						break;

					case 'appError':
						setError(message.data.error);
						break;
				}
			} catch {}
		};

		window.addEventListener('message', onMessage);
		return () => window.removeEventListener('message', onMessage);
	}, [router, setTitle]);

	React.useEffect(() => {
		if (!id) return;
		setTitle(id);

		if (!iframeRef.current) return;
		const iframe = iframeRef.current;

		const listener = () => {
			if (iframe.contentDocument) {
				setTitle(iframe.contentDocument.title);
			}
		};

		iframe.addEventListener('load', listener);
		return () => iframe.removeEventListener('load', listener);
	}, [id, setTitle]);

	return (
		<div className={'full col'}>
			{error && (
				<div style={{ width: '100%', padding: '1rem' }}>
					<Alert variant="error" title={`The '${id}' app has crashed.`}>
						<pre style={{ marginBottom: '1rem' }}>
							<code>{error}</code>
						</pre>

						<Button variant="primary" onClick={() => setError(null)}>
							Reload app
						</Button>
					</Alert>
				</div>
			)}
			{!!id && !error && (
				<iframe
					ref={iframeRef}
					src={`http://localhost:9010/app-resources/${id}/base.html`}
					style={{ border: '0', flexGrow: 1 }}
				/>
			)}
		</div>
	);
}

export function AppWindow(props: Props) {
	return <AppWindowContent key={props.id} {...props} />;
}
