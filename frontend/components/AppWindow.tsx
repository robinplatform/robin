type Props = {
	id: string;
};

export function AppWindow({ id }: Props) {
	return (
		<>
			<iframe src={`http://localhost:9010/app-resources/html/${id}`}></iframe>
		</>
	);
}
