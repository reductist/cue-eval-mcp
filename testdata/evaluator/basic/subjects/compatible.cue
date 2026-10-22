package basic

#Candidate: {
	id:     string & =~"^record-"
	nested: count: int & <=5
}
