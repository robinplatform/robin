import { getConfig } from '@robin/toolkit';
import { useQuery } from 'react-query';

export default function Home() {
	const { data: config } = useQuery({
		queryKey: ['getConfig'],
		queryFn: getConfig,
	});

	return <div>Hello world!</div>;
}
