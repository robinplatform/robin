import { useRouter } from 'next/router';
import React from 'react';
import { AppWindow } from '../../components/AppWindow';

export default function Page() {
	const router = useRouter();

	const id = `${router.query['id']}`;

	return <AppWindow id={id} />;
}
