import { useRouter } from 'next/router';
import React, { useReducer } from 'react';

type Props = {
	id: string | undefined;
	setTitle: (title: string) => void;
};

export function AppWindow({ id, setTitle }: Props) {
	const router = useRouter();

	const iframeRef = React.useRef<HTMLIFrameElement | null>(null);

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
						setTitle(message.data.title);
						break;
				}
			} catch {}
		};

		window.addEventListener('message', onMessage);
		return () => window.removeEventListener('message', onMessage);
	}, []);

	React.useEffect(() => {
		if (!id) return;
		if (!iframeRef.current) return;

		const iframe = iframeRef.current;

		const listener = (evt: unknown) => {
			if (iframe.contentDocument) {
				setTitle(iframe.contentDocument.title);
			}
		};

		iframe.addEventListener('load', listener);
		return () => iframe.removeEventListener('load', listener);
	}, [id]);

	return (
		<div className={'full col'}>
			{!!id && (
				<iframe
					ref={iframeRef}
					className={''}
					src={`http://localhost:9010/app-resources/${id}/base.html`}
					style={{ border: '0', flexGrow: 1 }}
				></iframe>
			)}
		</div>
	);
}
