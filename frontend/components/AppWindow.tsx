type Props = {
	id: string;
};

export function AppWindow({ id }: Props) {
	return (
		<>
			<iframe
				className={''}
				src={`http://localhost:9010/app-resources/html/${id}`}
				style={{ border: '0', flexGrow: 1 }}
			></iframe>
		</>
	);
}
